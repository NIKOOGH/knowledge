package com.mini.rpc.protocol;

import com.mini.rpc.serialization.Serializer;
import com.mini.rpc.serialization.SerializerFactory;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.MessageToByteEncoder;

/**
 * 协议编码器：Java对象 -> 自定义二进制帧
 *
 * 说明：
 * 1. 编码侧不存在粘包问题（发送时一对象一帧完整写出），
 *    粘包只发生在接收方聚合视角 —— 所以解码器才是重点
 * 2. MessageToByteEncoder 会自动 release 写入用的临时缓冲区
 */
public class ProtocolEncoder extends MessageToByteEncoder<RpcProtocolHolder> {

    private final SerializerFactory serializerFactory;

    public ProtocolEncoder(SerializerFactory serializerFactory) {
        this.serializerFactory = serializerFactory;
    }

    @Override
    protected void encode(ChannelHandlerContext ctx,
                          RpcProtocolHolder holder,
                          ByteBuf out) {
        // 选出协议头中声明的序列化器
        Serializer serializer = serializerFactory.getSerializer(holder.getSerializeType());
        // 复用 RpcProtocol 的写入逻辑，保证两端报文格式一致（单一事实来源）
        holder.getProtocol().encode(out, serializer, holder.getPayload());
    }
}
