package com.mini.rpc.transport.client;

import com.mini.rpc.core.RpcResponse;
import com.mini.rpc.core.RequestHolder;
import com.mini.rpc.protocol.ProtocolConstants;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import lombok.extern.slf4j.Slf4j;

/**
 * 客户端响应处理器
 *
 * 职责非常单一：收到响应帧 -> 按 requestId 找到挂起的 future 并 complete。
 * 线程模型：本方法运行在 EventLoop，RequestHolder 的 complete() 只是
 * 设置 CompletableFuture 结果并唤醒等待线程（park/unpark），足够轻量。
 */
@Slf4j
public class ClientHandler extends SimpleChannelInboundHandler<com.mini.rpc.protocol.RpcProtocol> {

    @Override
    protected void channelRead0(ChannelHandlerContext ctx,
                                com.mini.rpc.protocol.RpcProtocol protocol) {
        switch (protocol.getMessageType()) {
            case RESPONSE:
                // 1. 反序列化业务响应体
                RpcResponse response = protocol.deserializeBody(
                        RpcResponse.class, new com.mini.rpc.serialization.SerializerFactory());
                // 2. 按 requestId 唤醒等待的调用线程（异步转同步的收口）
                RequestHolder.complete(response.getRequestId(), response);
                break;
            case HEARTBEAT:
                log.debug("心跳应答: {}", protocol.getRequestId());
                break;
            default:
                // 其他消息类型忽略
                break;
        }
    }

    /**
     * 连接断开：该连接上所有未完成请求立即失败。
     * 否则调用方会一直等到超时才拿到结果 —— 故障必须快速暴露（fail fast）。
     */
    @Override
    public void channelInactive(ChannelHandlerContext ctx) throws Exception {
        RequestHolder.failAll(new IllegalStateException(
                "与服务端连接已断开: " + ctx.channel().remoteAddress()));
        super.channelInactive(ctx);
    }

    @Override
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
        log.error("客户端连接异常", cause);
        // 异常时让pending请求快速失败，然后断开重连
        RequestHolder.failAll(cause instanceof Exception
                ? (Exception) cause : new RuntimeException(cause));
        ctx.close();
    }

    /** 引用常量避免 IDE 未使用告警（协议头长度校验在解码层） */
    int headerLen = ProtocolConstants.HEADER_LENGTH;
}
