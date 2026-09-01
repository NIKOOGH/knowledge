package com.mini.rpc.protocol;

import com.mini.rpc.serialization.SerializerFactory;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.ByteToMessageDecoder;

import java.util.List;

/**
 * 协议解码器（核心：解决粘包/拆包）
 *
 * Netty 的 ByteToMessageDecoder 本身就是"积累式解码器"：
 * 1. 每次收到 TCP 数据先追加进内部累积缓冲区 cumulation
 * 2. 循环调用本类的 decode()，由我们决定何时消费字节
 * 3. decode() 返回后若 readerIndex 未推进，Netty 认为数据不足，会保留剩余字节等下次
 *
 * 这正是处理拆包的标准姿势；而粘包则是通过"按 length 精确读走一帧"
 * 让下一帧自然留给下一次循环来解决。
 */
public class ProtocolDecoder extends ByteToMessageDecoder {

    private final SerializerFactory serializerFactory;

    public ProtocolDecoder(SerializerFactory serializerFactory) {
        this.serializerFactory = serializerFactory;
    }

    @Override
    protected void decode(ChannelHandlerContext ctx, ByteBuf in, List<Object> out) {
        // 循环解码：一次 read 事件中可能包含多条完整消息（粘包），while保证全部解出
        while (in.readableBytes() >= ProtocolConstants.HEADER_LENGTH) {
            try {
                RpcProtocol protocol = RpcProtocol.decode(in, serializerFactory);
                if (protocol == null) {
                    // 字节不够一帧（拆包），跳出等待更多数据
                    break;
                }
                // 一帧解出，交给 pipeline 后续 handler 处理
                out.add(protocol);

                // 心跳消息直接在解码层响应，不进入业务处理（可选优化点）
                if (protocol.getMessageType() == MessageType.HEARTBEAT) {
                    // 简单回一个心跳应答，保持双向活性
                    ctx.writeAndFlush(buildHeartbeatAck(protocol.getRequestId()));
                }
            } catch (Exception e) {
                // 解码异常 = 报文不可信，关闭连接避免解码器状态污染导致雪崩
                ctx.close();
                throw e;
            }
        }
    }

    private RpcProtocol buildHeartbeatAck(long requestId) {
        RpcProtocol ack = new RpcProtocol();
        ack.setMessageType(MessageType.HEARTBEAT);
        ack.setRequestId(requestId);
        return ack;
    }
}
