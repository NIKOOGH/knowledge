package com.mini.rpc.example;

import com.mini.rpc.loadbalance.LoadBalancer;
import com.mini.rpc.loadbalance.RandomLoadBalancer;
import com.mini.rpc.loadbalance.RoundRobinLoadBalancer;
import com.mini.rpc.loadbalance.ConsistentHashLoadBalancer;
import com.mini.rpc.proxy.RpcClientProxy;
import com.mini.rpc.registry.NacosServiceRegistry;
import com.mini.rpc.transport.client.NettyRpcClient;

/**
 * 服务消费者启动示例
 *
 * 展示三种负载均衡的切换，并像调用本地方法一样发起远程调用。
 */
public class ConsumerDemo {

    public static void main(String[] args) throws Exception {
        String nacosAddr = "127.0.0.1:8848";

        // 1. 初始化注册中心客户端
        NacosServiceRegistry registry = new NacosServiceRegistry(nacosAddr);

        // 2. 订阅目标服务：首次拉取 + 变更推送刷新本地缓存
        registry.subscribeAndCache(HelloService.class.getName(), (name, list) ->
                System.out.println("[订阅回调] " + name + " 实例数=" + list.size()));

        // 3. 组装客户端 + 选择负载均衡策略
        NettyRpcClient client = new NettyRpcClient();
        LoadBalancer lb = pickBalancer(args); // 默认随机；可用参数指定

        // 4. 生成远程调用代理 —— 业务代码此刻只依赖接口
        RpcClientProxy proxy = new RpcClientProxy(registry, client, lb);
        HelloService hello = proxy.getProxy(HelloService.class);

        // 5. 发起调用：与本地方法调用零区别，网络细节全部被框架屏蔽
        System.out.println(hello.sayHi("mini-rpc"));
        System.out.println("1 + 2 = " + hello.add(1, 2));

        // 连续调用观察轮询/一致性哈希分流效果（可启动多个Provider端口做实验）
        for (int i = 0; i < 6; i++) {
            System.out.println(i + " -> " + hello.sayHi("u" + i));
            Thread.sleep(200);
        }

        client.close();
    }

    /**
     * 通过启动参数选择LB策略：
     * java ConsumerDemo [random|roundrobin|hash]
     */
    private static LoadBalancer pickBalancer(String[] args) {
        String type = args.length > 0 ? args[0].toLowerCase() : "random";
        if ("roundrobin".equals(type)) {
            System.out.println("负载均衡策略: 平滑加权轮询");
            return new RoundRobinLoadBalancer();
        }
        if ("hash".equals(type)) {
            System.out.println("负载均衡策略: 一致性哈希");
            return new ConsistentHashLoadBalancer();
        }
        System.out.println("负载均衡策略: 随机");
        return new RandomLoadBalancer();
    }
}
