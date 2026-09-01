package com.mini.rpc.registry;

import com.alibaba.nacos.api.NacosFactory;
import com.alibaba.nacos.api.PropertyKeyConst;
import com.alibaba.nacos.api.exception.NacosException;
import com.alibaba.nacos.api.naming.NamingService;
import com.alibaba.nacos.api.naming.listener.EventListener;
import com.alibaba.nacos.api.naming.listener.NamingEvent;
import com.alibaba.nacos.api.naming.pojo.Instance;
import lombok.extern.slf4j.Slf4j;

import java.net.InetSocketAddress;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Properties;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.function.BiConsumer;

/**
 * 基于 Nacos 的服务注册与发现实现
 *
 * 核心机制：
 * 1. 注册（Provider侧）：naming.registerInstance(服务名, ip, port)
 *    Nacos 客户端自动维护 5s 心跳，15s 未收到标记不健康，30s 剔除 —— 临时实例默认行为
 * 2. 发现（Consumer侧）：naming.getAllInstances(服务名) 拉取 + 订阅变更推送
 *    订阅回调里更新本地缓存，调用方读取时零网络开销（客户端软负载的标准做法）
 */
@Slf4j
public class NacosServiceRegistry {

    /** 复用同一个 NamingService 连接（底层gRPC长连接） */
    private final NamingService namingService;

    /**
     * 本地服务列表缓存：serviceName -> 该服务的可用节点列表
     * 消费端热路径直接读这里，不会每次调用都请求Nacos
     */
    private static final Map<String, List<InetSocketAddress>> LOCAL_CACHE =
            new ConcurrentHashMap<>();

    public NacosServiceRegistry(String serverAddr) {
        Properties props = new Properties();
        props.setProperty(PropertyKeyConst.SERVER_ADDR, serverAddr);
        try {
            this.namingService = NacosFactory.createNamingService(props);
        } catch (NacosException e) {
            throw new RuntimeException("初始化Nacos失败: " + serverAddr, e);
        }
    }

    // ==================== Provider：服务注册 ====================

    /**
     * 注册单个服务实例
     *
     * @param serviceName 接口全限定名
     * @param host        服务地址
     * @param port        服务端口
     */
    public void register(String serviceName, String host, int port) {
        Instance instance = new Instance();
        instance.setIp(host);
        instance.setPort(port);

        // 附加元数据：负载均衡权重、集群等可放这里
        instance.setWeight(1.0);
        // ephemeral=true 临时实例：会话断开自动剔除，天然支持故障摘除
        instance.setEphemeral(true);

        try {
            namingService.registerInstance(serviceName, instance);
            log.info("服务注册成功: {} -> {}:{}", serviceName, host, port);
        } catch (NacosException e) {
            throw new RuntimeException("服务注册失败: " + serviceName, e);
        }
    }

    // ==================== Consumer：服务发现 + 变更订阅 ====================

    /**
     * 拉取服务列表并订阅变更（消费端启动时对每个依赖服务调用一次）
     *
     * 双保险策略：
     * - 首次同步拉取，保证启动即可用
     * - EventListener 收到推送后刷新本地缓存，之后读缓存
     */
    public void subscribeAndCache(String serviceName,
                                  BiConsumer<String, List<InetSocketAddress>> onUpdate) {
        try {
            // 1. 首次拉取并缓存
            refreshCache(serviceName);
            onUpdate.accept(serviceName, getInstances(serviceName));

            // 2. 订阅后续变更（Nacos UDP/gRPC 推送）
            namingService.subscribe(serviceName, (EventListener) event -> {
                if (event instanceof NamingEvent) {
                    NamingEvent namingEvent = (NamingEvent) event;
                    log.info("服务实例变更: {}, 新列表={}", serviceName, namingEvent.getInstances());
                    try {
                        refreshCache(serviceName);       // 拉全量重建缓存
                        onUpdate.accept(serviceName, getInstances(serviceName));
                    } catch (NacosException e) {
                        // 监听回调不能抛受检异常，记日志保留旧缓存即可
                        log.error("刷新服务缓存失败: {}", serviceName, e);
                    }
                }
            });
        } catch (NacosException e) {
            throw new RuntimeException("服务发现失败: " + serviceName, e);
        }
    }

    /**
     * 从 Nacos 拉取健康实例并写入本地缓存
     */
    private void refreshCache(String serviceName) throws NacosException {
        // onlyHealthyInstance=true 只保留心跳健康的节点，故障自动摘除
        List<Instance> instances = namingService.selectInstances(serviceName, true);

        List<InetSocketAddress> addresses = instances.stream()
                .map(i -> new InetSocketAddress(i.getIp(), i.getPort()))
                .collect(java.util.stream.Collectors.toCollection(CopyOnWriteArrayList::new));

        LOCAL_CACHE.put(serviceName, addresses);
    }

    /**
     * 获取某服务的当前可用节点列表（消费者调用前使用）
     */
    public List<InetSocketAddress> getInstances(String serviceName) {
        return LOCAL_CACHE.getOrDefault(serviceName, Collections.emptyList());
    }

    /**
     * 服务优雅下线：注销实例而非直接杀进程，让流量先切走
     */
    public void deregister(String serviceName, String host, int port) {
        try {
            namingService.deregisterInstance(serviceName, host, port);
            log.info("服务下线成功: {} {}:{}", serviceName, host, port);
        } catch (NacosException e) {
            log.error("服务下线失败: {}", serviceName, e);
        }
    }

    // 其他公共方法（注册/注销接口委托给内部实现）
}
