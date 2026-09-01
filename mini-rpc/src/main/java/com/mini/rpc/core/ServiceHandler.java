package com.mini.rpc.core;

import lombok.extern.slf4j.Slf4j;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 服务端：接口名 -> 实现实例 的映射表
 *
 * Provider 启动时调用 publish() 发布服务；
 * ServerHandler 收到请求后按 interfaceName 反查实现类做反射调用。
 */
@Slf4j
public final class ServiceHandler {

    private ServiceHandler() {
    }

    /** 接口全限定名 -> 服务实例（服务端可发布多个服务） */
    private static final Map<String, Object> SERVICE_MAP = new ConcurrentHashMap<>();

    /** 服务端已处理请求计数（观测用） */
    public static final AtomicLong REQUEST_COUNTER = new AtomicLong();

    /**
     * 发布服务到本地映射表
     */
    public static void publish(String interfaceName, Object instance) {
        Object prev = SERVICE_MAP.putIfAbsent(interfaceName, instance);
        if (prev != null) {
            throw new IllegalStateException("服务重复注册: " + interfaceName);
        }
        log.info("服务已发布: {} -> {}", interfaceName, instance.getClass().getName());
    }

    /**
     * 查找服务实现
     */
    public static Object lookup(String interfaceName) {
        Object service = SERVICE_MAP.get(interfaceName);
        if (service == null) {
            throw new IllegalStateException("未找到服务实现: " + interfaceName);
        }
        return service;
    }
}
