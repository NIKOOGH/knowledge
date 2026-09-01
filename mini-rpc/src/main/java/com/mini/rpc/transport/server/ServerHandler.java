package com.mini.rpc.transport.server;

import com.mini.rpc.core.RpcRequest;
import com.mini.rpc.core.RpcResponse;
import com.mini.rpc.core.ServiceHandler;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import lombok.extern.slf4j.Slf4j;

import java.lang.reflect.Method;

/**
 * 服务端业务处理器
 *
 * 关键点：
 * 1. 继承 SimpleChannelInboundHandler<RpcProtocol>：
 *    - 自动按消息类型分发到 channelRead0
 *    - 处理完自动 release 消息对象，防内存泄漏
 * 2. 业务线程池隔离（重要！）：
 *    Netty EventLoop 只应做 IO，反射调用可能很慢（DB/下游），
 *    直接在 EventLoop 执行会阻塞同一 NIO 线程上的所有连接 —— 经典事故点。
 */
@Slf4j
public class ServerHandler extends SimpleChannelInboundHandler<com.mini.rpc.protocol.RpcProtocol> {

    /** 业务执行线程池：与 IO 线程解耦 */
    private static final java.util.concurrent.ThreadPoolExecutor BUSINESS_POOL =
            new java.util.concurrent.ThreadPoolExecutor(
                    16, 32, 60, java.util.concurrent.TimeUnit.SECONDS,
                    new java.util.concurrent.LinkedBlockingQueue<>(1000),
                    r -> new Thread(r, "rpc-biz-" + BusinessThreadSeq.next()),
                    new java.util.concurrent.ThreadPoolExecutor.CallerRunsPolicy());

    @Override
    protected void channelRead0(ChannelHandlerContext ctx,
                                com.mini.rpc.protocol.RpcProtocol protocol) {
        // 心跳包已在解码层处理，这里只关心请求
        if (protocol.getMessageType() != com.mini.rpc.protocol.MessageType.REQUEST) {
            return;
        }

        // ---- 反序列化请求体（也可以放业务池内做，取决于开销权衡）----
        RpcRequest request = protocol.deserializeBody(
                RpcRequest.class, new com.mini.rpc.serialization.SerializerFactory());

        // ---- 投递到业务线程池异步执行，EventLoop 立即返回继续 read ----
        BUSINESS_POOL.execute(() -> {
            RpcResponse response = invokeService(request);
            // 通过原 Channel 回写响应（编解码器挂在 pipeline 上自动做协议转换）
            writeResponse(ctx, protocol.getRequestId(), response);
        });
    }

    /**
     * 反射调用真实服务（RPC服务端最核心的一步）
     */
    private RpcResponse invokeService(RpcRequest request) {
        try {
            ServiceHandler.REQUEST_COUNTER.incrementAndGet();

            // 1. 接口名 -> 实现实例
            Object service = ServiceHandler.lookup(request.getInterfaceName());

            // 2. 按 方法名+参数类型 定位 Method（正确处理重载）
            Method method = service.getClass().getMethod(
                    request.getMethodName(), request.getParameterTypes());

            // 3. 反射调用并包装返回值
            Object result = method.invoke(service, request.getParameters());
            return RpcResponse.success(request.getRequestId(), result);

        } catch (java.lang.reflect.InvocationTargetException e) {
            // 目标方法内部抛出的业务异常：取原始异常
            Throwable cause = e.getTargetException();
            log.error("业务异常: {}.{}", request.getInterfaceName(),
                    request.getMethodName(), cause);
            return RpcResponse.fail(request.getRequestId(), cause.toString());
        } catch (Exception e) {
            log.error("调用失败: {}", request.getMethodName(), e);
            return RpcResponse.fail(request.getRequestId(), e.toString());
        }
    }

    private void writeResponse(ChannelHandlerContext ctx, long requestId, RpcResponse response) {
        com.mini.rpc.serialization.SerializerFactory factory =
                new com.mini.rpc.serialization.SerializerFactory();
        com.mini.rpc.serialization.Serializer serializer = factory.getDefault();

        com.mini.rpc.protocol.RpcProtocol resp = new com.mini.rpc.protocol.RpcProtocol();
        resp.setMessageType(com.mini.rpc.protocol.MessageType.RESPONSE);
        resp.setRequestId(requestId);

        // 编码器会完成序列化+长度回填；Holder 负责传递载荷
        ctx.writeAndFlush(new com.mini.rpc.protocol.RpcProtocolHolder(resp, response, serializer.type()));
    }

    /** 连接断开时释放资源（如有每连接状态需清理） */
    @Override
    public void channelInactive(ChannelHandlerContext ctx) throws Exception {
        log.debug("客户端断开: {}", ctx.channel().remoteAddress());
        super.channelInactive(ctx);
    }

    /** 异常兜底：打印并关闭连接，避免半死连接堆积 */
    @Override
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
        log.error("连接异常: {}", ctx.channel().remoteAddress(), cause);
        ctx.close();
    }

    /** 线程名序号（仅命名用） */
    private static class BusinessThreadSeq {
        private static final java.util.concurrent.atomic.AtomicInteger SEQ = new java.util.concurrent.atomic.AtomicInteger(0);
        static int next() { return SEQ.incrementAndGet(); }
    }
}
