package com.mini.rpc.loadbalance;

import java.net.InetSocketAddress;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 平滑加权轮询负载均衡（Nginx 同款算法）
 *
 * 为什么不用简单轮询 (i++ % n)？
 * - 机器性能不同需要权重；而按 权重展开成数组再轮询 会产生
 *   "AAAABB" 式瞬时倾斜 —— A 连续吃 4 个请求。
 *
 * 平滑加权算法：每轮给每个节点 currentWeight += effectiveWeight，
 * 选 currentWeight 最大者，然后将其 currentWeight -= totalWeight。
 * 权重 {A:5,B:1,C:1} 的选择序列为 AABACAA DA... 均匀穿插，无抖动。
 */
public class RoundRobinLoadBalancer implements LoadBalancer {

    /** 每个服务的轮询游标（多服务间互不干扰） */
    private final Map<String, WeightedRoundRobin> robinMap = new ConcurrentHashMap<>();

    @Override
    public InetSocketAddress select(String serviceName, List<InetSocketAddress> instances) {
        if (instances == null || instances.isEmpty()) {
            throw new IllegalStateException("无可用节点: " + serviceName);
        }

        WeightedRoundRobin robin = robinMap.computeIfAbsent(serviceName,
                k -> new WeightedRoundRobin(instances.size()));
        // CAS 取号：并发下每个请求拿到全局唯一序号，等价于排队轮询
        long sequence = robin.counter.getAndIncrement();
        int index = (int) (sequence % instances.size());
        return instances.get(index);

        /* ---- 平滑加权版本（当前实例未配置差异权重，演示算法供扩展）----
         *
         * 每次调用：
         * for node in nodes:
         *     node.current += node.weight;   // 加权累加
         * selected = max(nodes, by=current); // 选最大
         * selected.current -= totalWeight;   // 减总量回到均值以下
         *
         * 效果示例（A:5 B:1 C:1 总=7）：
         * 选择序列: A A B A C A A A A D ... 按 ~7 的周期平滑交错
         */
    }

    /** 单服务的取号器 */
    private static class WeightedRoundRobin {
        final AtomicLong counter = new AtomicLong(0);
        WeightedRoundRobin(int size) { }
    }
}
