package com.mini.rpc.loadbalance;

import java.net.InetSocketAddress;
import java.util.List;
import java.util.concurrent.ThreadLocalRandom;

/**
 * 随机负载均衡
 *
 * 实现最简单，大量请求下统计上等价于均匀分流。
 * 用 ThreadLocalRandom 而非共享 Random：后者 CAS 自旋在多线程下发会后成为瓶颈。
 */
public class RandomLoadBalancer implements LoadBalancer {

    @Override
    public InetSocketAddress select(String serviceName, List<InetSocketAddress> instances) {
        if (instances == null || instances.isEmpty()) {
            throw new IllegalStateException("无可用节点: " + serviceName);
        }
        int index = ThreadLocalRandom.current().nextInt(instances.size());
        return instances.get(index);
    }
}
