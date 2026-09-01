package com.mini.rpc.loadbalance;

import java.net.InetSocketAddress;
import java.util.List;

/**
 * 负载均衡策略接口（策略模式）
 *
 * 输入：注册中心拉到的健康节点列表 + 服务名
 * 输出：本次调用选中的节点
 *
 * 客户端软负载：LB 逻辑在消费端进程内，无需独立 LB 中间件，
 * 省一次网络转发，且可结合熔断等本地状态做更智能路由。
 */
@FunctionalInterface
public interface LoadBalancer {

    /**
     * 从候选列表中选择一个节点
     *
     * @param serviceName 服务名（一致性哈希需要用它隔离哈希环）
     * @param instances   健康实例列表
     * @return 选中的节点
     */
    InetSocketAddress select(String serviceName, List<InetSocketAddress> instances);
}
