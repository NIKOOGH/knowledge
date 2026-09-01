package com.mini.rpc.core;

import io.netty.channel.Channel;
import lombok.extern.slf4j.Slf4j;

import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 请求-响应配对容器（异步转同步的核心组件）
 *
 * 原理：
 * 发送请求前：requestId -> CompletableFuture 注册到本Map
 * 收到响应后：按 requestId 取出 Future 并 complete()，调用线程 get() 返回
 *
 * 这就是 Netty 异步 IO 变成同步 RPC 语义的桥梁，
 * Dubbo 中对应 DefaultFuture 的简化版。
 */
@Slf4j
public final class RequestHolder {

    private RequestHolder() {
    }

    /** requestId -> 未完成的future */
    private static final Map<Long, CompletableFuture<Object>> PENDING =
            new ConcurrentHashMap<>();

    /**
     * 客户端发送前注册
     */
    public static void register(long requestId, CompletableFuture<Object> future) {
        PENDING.put(requestId, future);
    }

    /**
     * 响应到达时完成对应 future（在 Netty EventLoop 线程回调）
     */
    public static void complete(long requestId, Object result) {
        CompletableFuture<Object> future = PENDING.remove(requestId);
        if (future != null) {
            future.complete(result);
        } else {
            // 已超时移除后迟到的响应，直接丢弃并告警
            log.warn("收到迟到/未知响应，已丢弃 requestId={}", requestId);
        }
    }

    /**
     * 超时或连接断开时清理并使 future 异常完成，防止调用线程永久挂起
     */
    public static void removeAndFail(long requestId, Throwable cause) {
        CompletableFuture<Object> future = PENDING.remove(requestId);
        if (future != null) {
            future.completeExceptionally(cause);
        }
    }

    /**
     * 连接关闭时批量失败该 Channel 上未完成的请求（简化：遍历全部）
     * 生产实现应为每个 Channel 维护独立 pending 队列
     */
    public static void failAll(Throwable cause) {
        PENDING.values().forEach(f -> f.completeExceptionally(cause));
        PENDING.clear();
    }
}
