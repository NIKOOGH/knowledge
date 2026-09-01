package com.mini.rpc.loadbalance;

import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 一致性哈希负载均衡
 *
 * 核心思想：
 * 把节点和请求都映射到 [0, 2^32) 哈希环上，请求落在环上后顺时针找到第一个节点。
 *
 * 为什么需要"虚拟节点"？
 * 若只有3个真实节点直接哈希上环，各段弧长严重不均（数据倾斜）。
 * 给每个真实节点生成 VIRTUAL_NUM 个虚拟节点均匀铺满哈希环，
 * 真实节点承担的区间趋于均匀。
 *
 * 关键特性（相对取模 hash % n 的优势）：
 * 节点增减只影响相邻区段的流量分布，其余节点映射关系不变 —— 会话亲和场景友好。
 */
public class ConsistentHashLoadBalancer implements LoadBalancer {

    /** 每个真实节点的虚拟节点数量：越大分布越均匀，TreeMap 也越大 */
    private static final int VIRTUAL_NODES = 150;

    /**
     * 服务名 -> 哈希环
     * TreeMap: 按hash升序排列的虚拟节点表；
     * ceilingEntry(hash) 正是"顺时针找第一个>=该hash的节点"，O(log n)
     */
    private final Map<String, TreeMap<Long, InetSocketAddress>> ringMap = new ConcurrentHashMap<>();

    @Override
    public InetSocketAddress select(String serviceName, List<InetSocketAddress> instances) {
        if (instances == null || instances.isEmpty()) {
            throw new IllegalStateException("无可用节点: " + serviceName);
        }

        // 获取/按需重建该服务的哈希环（节点数变化作为简化指纹，生产建议完整比对版本）
        TreeMap<Long, InetSocketAddress> ring = buildRingIfChanged(serviceName, instances);

        if (ring == null || ring.isEmpty()) {
            throw new IllegalStateException("哈希环为空: " + serviceName);
        }

        /**
         * 选择依据 = 当前线程id（示例）。
         * 生产中常用 参数对象.toString() 或 用户ID 做key，
         * 实现"同一用户固定路由到同一节点"的会话亲和。
         */
        long requestHash = murmurOrSimpleHash(
                Thread.currentThread().getName() + System.nanoTime());

        // 顺时针找第一个 >= requestHash 的虚拟节点
        Map.Entry<Long, InetSocketAddress> entry = ring.ceilingEntry(requestHash);
        if (entry == null) {
            // 环绕到首个节点（回环）
            entry = ring.firstEntry();
        }
        return entry.getValue();
    }

    /** 简化版变更检测与重建（以“节点数”为指纹；生产建议完整比对） */
    private TreeMap<Long, InetSocketAddress> buildRingIfChanged(String serviceName,
                                                                List<InetSocketAddress> instances) {
        TreeMap<Long, InetSocketAddress> existing = ringMap.get(serviceName);
        int existingRealNodes = existing == null ? -1 : countUnique(existing.values());
        if (existing == null || existingRealNodes != countUnique(instances)) {
            TreeMap<Long, InetSocketAddress> ring = buildRing(serviceName, instances);
            ringMap.put(serviceName, ring);
            return ring;
        }
        return existing; // 未变化，直接复用现有哈希环
    }

    /** 构建哈希环：为每个真实节点创建 VIRTUAL_NODES 个虚拟节点散列上环 */
    private TreeMap<Long, InetSocketAddress> buildRing(String key,
                                                       List<InetSocketAddress> instances) {
        TreeMap<Long, InetSocketAddress> ring = new TreeMap<>();
        for (InetSocketAddress node : instances) {
            for (int i = 0; i < VIRTUAL_NODES; i++) {
                // 虚拟节点命名 "ip:port#VN{i}"，其 hash 散列更随机均匀
                String vnodeKey = node.getAddress().getHostAddress()
                        + ":" + node.getPort() + "#VN" + i;
                ring.put(murmurOrSimpleHash(vnodeKey), node); // value指向真实节点
            }
        }
        return ring;
    }

    private int countUnique(Iterable<InetSocketAddress> values) {
        java.util.Set<String> set = new java.util.HashSet<>();
        values.forEach(a -> set.add(a.getHostString() + ":" + a.getPort()));
        return set.size();
    }

    /**
     * FNV-1a 哈希：简单、分布性好、无需引入依赖
     * 生产可用 MurmurHash / xxHash 等
     */
    private static long murmurOrSimpleHash(String key) {
        final int p = 16777619;
        int hash = (int) 2166136261L;
        for (byte b : key.getBytes(StandardCharsets.UTF_8)) {
            hash = (hash ^ b) * p;
        }
        // 强制非负（Java 无无符号int，掉符号位会破坏环形语义）
        return hash & 0x7fffffffL;
    }
}
