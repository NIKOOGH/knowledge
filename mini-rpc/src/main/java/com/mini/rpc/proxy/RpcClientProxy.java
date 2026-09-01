package com.mini.rpc.proxy;

import com.mini.rpc.circuitbreaker.CircuitBreakerManager;
import com.mini.rpc.core.RpcRequest;
import com.mini.rpc.core.RpcResponse;
import com.mini.rpc.loadbalance.LoadBalancer;
import com.mini.rpc.loadbalance.RandomLoadBalancer;
import com.mini.rpc.registry.NacosServiceRegistry;
import com.mini.rpc.transport.client.NettyRpcClient;
import lombok.extern.slf4j.Slf4j;

import java.lang.reflect.InvocationHandler;
import java.lang.reflect.Method;
import java.lang.reflect.Proxy;
import java.net.InetSocketAddress;
import java.util.List;

/**
 * 客户端动态代理 + 远程调用编排入口
 *
 * 这是整个 RPC 框架的"门面"：把 注册发现 → 负载均衡 → 熔断 → 传输 串成一条链路。
 *
 * 为什么用 JDK 动态代理？
 * 业务方拿到的是 HelloService 接口对象，调用 sayHi() 时会被 InvocationHandler 拦截，
 * 将"本地方法调用"偷梁换柱为"网络远程调用" —— 这就是 Dubbo @Reference 的本质。
 */
@Slf4j
public class RpcClientProxy implements InvocationHandler {

    /** 默认请求超时：网络抖动/服务端阻塞的保护阀门 */
    private static final long DEFAULT_TIMEOUT_MS = 3000;

    private final NacosServiceRegistry registry;   // 服务发现（含本地缓存）
    private final NettyRpcClient client;           // 网络传输
    private final LoadBalancer loadBalancer;       // 负载均衡策略（可替换）
    private final CircuitBreakerManager breakers;  // 节点级熔断器

    public RpcClientProxy(NacosServiceRegistry registry,
                          NettyRpcClient client,
                          LoadBalancer loadBalancer) {
        this.registry = registry;
        this.client = client;
        this.loadBalancer = loadBalancer != null ? loadBalancer : new RandomLoadBalancer();
        this.breakers = new CircuitBreakerManager();
    }

    /**
     * 生成接口的远程调用桩(Stub)
     * 使用方：HelloService svc = proxy.getProxy(HelloService.class);
     */
    @SuppressWarnings("unchecked")
    public <T> T getProxy(Class<T> interfaceClass) {
        return (T) Proxy.newProxyInstance(
                interfaceClass.getClassLoader(),
                new Class<?>[]{interfaceClass},
                this);
    }

    /**
     * 方法调用拦截点 —— 一切远程调用的起点
     */
    @Override
    public Object invoke(Object proxy, Method method, Object[] args) throws Throwable {
        // Object 自身方法（toString/hashCode等）直接本地执行，不做远程化
        if (method.getDeclaringClass() == Object.class) {
            return method.invoke(this, args);
        }

        // ========== 1. 构造 RpcRequest ==========
        String serviceName = method.getDeclaringClass().getName(); // 接口全限定名 = 服务键
        RpcRequest request = new RpcRequest();
        request.setRequestId(NettyRpcClient.nextRequestId());
        request.setInterfaceName(serviceName);
        request.setMethodName(method.getName());
        request.setParameterTypes(method.getParameterTypes());
        request.setParameters(args);

        // ========== 2. 服务发现：从注册中心缓存拉取健康节点列表 ==========
        List<InetSocketAddress> instances = registry.getInstances(serviceName);
        if (instances.isEmpty()) {
            throw new IllegalStateException("无可用服务实例: " + serviceName);
        }

        // ========== 3. 负载均衡 -> 选择目标节点（带重试，跳过熔断打开的节点） ==========
        InetSocketAddress address = selectWithCircuitBreaker(serviceName, instances);
        if (address == null) {
            // 所有节点都被熔断 => 整体降级失败。此处可插入 fallback 本地兜底逻辑
            throw new IllegalStateException("所有节点均已熔断, 执行降级: " + serviceName);
        }

        // ========== 4. 熔断准入 + 发起远程调用 ==========
        com.mini.rpc.circuitbreaker.CircuitBreaker breaker = breakers.get(
                address.getHostString(), address.getPort());
        if (!breaker.tryAcquire()) {
            throw new IllegalStateException("节点已熔断, 快速失败: " + address);
        }

        try {
            RpcResponse response = client.send(address, request, DEFAULT_TIMEOUT_MS);

            // 成功 -> 熔断器记成功（喂给滑动窗口）
            breakers.recordSuccess(address.getHostString(), address.getPort());

            if (response.hasException()) {
                // 服务端业务异常原样抛出给调用方（还原本地调用语义）
                throw new RuntimeException("远程业务异常: " + response.getExceptionMessage());
            }
            return response.getReturnValue();

        } catch (Exception e) {
            // 失败 -> 熔断器记失败（可能触发 OPEN）
            breakers.recordFailure(address.getHostString(), address.getPort());
            throw e;
        }
    }

    /**
     * 选节点：先按LB策略选，若该节点已熔断则从候选中剔除后重选。
     * 避免流量持续打向已OPEN节点的白白浪费。
     */
    private InetSocketAddress selectWithCircuitBreaker(String serviceName,
                                                       List<InetSocketAddress> instances) {
        List<InetSocketAddress> candidates = new java.util.ArrayList<>(instances);
        while (!candidates.isEmpty()) {
            InetSocketAddress picked = loadBalancer.select(serviceName, candidates);
            if (breakers.allowRequest(picked.getHostString(), picked.getPort())) {
                return picked;
            }
            log.warn("节点已熔断, 从候选剔除重选: {}", picked);
            candidates.remove(picked);
        }
        return null;
    }
}
