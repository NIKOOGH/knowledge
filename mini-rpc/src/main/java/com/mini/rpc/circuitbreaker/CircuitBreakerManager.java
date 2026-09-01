package com.mini.rpc.circuitbreaker;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 熔断器管理器：按"服务名+节点"粒度维护熔断器
 *
 * 为什么按节点而不是按服务？
 * 同一服务不同实例可能只有一台故障：按服务熔断会误伤健康节点（雪崩半径过大），
 * 按节点隔离则只屏蔽坏实例，这正是 Sentinel 集群流控/热点规则的思路之一。
 */
public class CircuitBreakerManager {

    /** key: "host:port" -> 该节点的独立熔断器 */
    private final Map<String, CircuitBreaker> breakers = new ConcurrentHashMap<>();

    /**
     * 获取(或惰性创建)某节点的熔断器
     * 默认参数：最少5个样本、失败率50%触发、OPEN持续10s
     */
    public CircuitBreaker get(String host, int port) {
        return breakers.computeIfAbsent(host + ":" + port,
                k -> new CircuitBreaker(5, 50, 10_000));
    }

    /**
     * 一站式判断：该节点当前是否允许发起调用
     */
    public boolean allowRequest(String host, int port) {
        return get(host, port).tryAcquire();
    }

    /** 记录成功（配对 allowRequest 使用） */
    public void recordSuccess(String host, int port) {
        get(host, port).onSuccess();
    }

    /** 记录失败 */
    public void recordFailure(String host, int port) {
        get(host, port).onError();
    }
}
