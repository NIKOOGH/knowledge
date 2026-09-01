package com.mini.rpc.protocol;

import com.mini.rpc.serialization.Serializer;
import com.mini.rpc.serialization.SerializerFactory;
import io.netty.buffer.ByteBuf;

/**
 * 消息头 + 消息体的完整协议对象（Value Object）
 *
 * 一个 RpcProtocol 实例即为一"帧"完整消息。
 * 编码：RpcProtocol -> ByteBuf；解码：ByteBuf -> RpcProtocol
 */
public class RpcProtocol {

    /** 协议版本 */
    private byte version = ProtocolConstants.VERSION;

    /** 消息类型（REQUEST / RESPONSE / HEARTBEAT） */
    private MessageType messageType;

    /** 序列化算法类型（JSON 等，见 SerializerFactory） */
    private byte serializeType;

    /**
     * 请求ID：全局唯一。
     * 核心作用：
     * 1. 一条连接上可以并发跑多个请求（多路复用），响应回来后靠它配对
     * 2. 客户端 RequestHolder 以 requestId -> RpcFuture 映射实现异步转同步
     */
    private long requestId;

    /** 消息体字节数（不含消息头），解决粘包/拆包的关键字段 */
    private int length;

    /** 消息体原始字节（已序列化） */
    private byte[] body;

    // ==================== 编码：对象 -> 字节 ====================

    /**
     * 将本协议对象写入 ByteBuf
     *
     * @param buf 目标缓冲区
     * @param serializer 序列化器（由 serializeType 决定）
     * @param payload 待序列化的业务对象（RpcRequest 或 RpcResponse）
     */
    public void encode(ByteBuf buf, Serializer serializer, Object payload) {
        this.body = serializer.serialize(payload);
        this.length = body == null ? 0 : body.length;
        this.serializeType = serializer.type();

        writeHeader(buf);
        if (length > 0) {
            buf.writeBytes(body);
        }
    }

    /**
     * 写入消息头（严格按照协议格式逐字段写入）
     */
    private void writeHeader(ByteBuf buf) {
        buf.writeInt(ProtocolConstants.MAGIC_NUMBER);   // 4B 魔数
        buf.writeByte(version);                          // 1B 版本
        buf.writeByte(messageType.getCode());           // 1B 消息类型
        buf.writeByte(serializeType);                    // 1B 序列化类型
        buf.writeLong(requestId);                        // 8B 请求ID
        buf.writeInt(length);                            // 4B 消息体长度
    }

    // ==================== 解码：字节 -> 对象 ====================

    /**
     * 尝试从 ByteBuf 中解出一帧完整消息（不消耗不够读时的字节 —— 积累式解码）
     *
     * @param in 读缓冲区
     * @return 解出的协议对象；字节不足返回 null（等待下一次数据到达继续解码）
     */
    public static RpcProtocol decode(ByteBuf in, SerializerFactory factory) {
        /* ---------- 第一步：校验可用字节数是否够一个消息头 ---------- */
        // 这就是解决"拆包"的第一层：头部都可能不完整，先攒着不读
        if (in.readableBytes() < ProtocolConstants.HEADER_LENGTH) {
            return null; // 返回null后Netty会保留缓冲区，等下一段数据来再重试
        }

        /* ---------- 第二步：标记读指针位置，先预览魔数 ---------- */
        in.markReaderIndex();

        int magic = in.readInt();
        // 魔数不匹配 => 不是本协议的流量（或流被污染），由上层关闭连接
        if (magic != ProtocolConstants.MAGIC_NUMBER) {
            throw new IllegalStateException("非法魔数: 0x" + Integer.toHexString(magic)
                    + "，疑似非RPC流量或遭受攻击，应断开连接");
        }

        /* ---------- 第三步：解析头部各字段 ---------- */
        RpcProtocol protocol = new RpcProtocol();
        protocol.version = in.readByte();
        protocol.messageType = MessageType.of(in.readByte());
        protocol.serializeType = in.readByte();
        protocol.requestId = in.readLong();
        protocol.length = in.readInt();

        /* ---------- 第四步：长度合法性检查（防畸形报文/OOM攻击） ---------- */
        if (protocol.length < 0 || protocol.length > ProtocolConstants.MAX_BODY_LENGTH) {
            throw new IllegalStateException("非法消息体长度: " + protocol.length);
        }

        /* ---------- 第五步：校验消息体是否已全部到达（粘包/拆包核心） ---------- */
        // readableBytes() < length 说明一条消息只到了一部分（拆包场景）
        if (in.readableBytes() < protocol.length) {
            // 重置读指针到消息开头，本次不解码；
            // 剩余字节仍在缓冲区中，待后续数据到达时从头重新解码整帧
            in.resetReaderIndex();
            return null;
        }

        /* ---------- 第六步：读取完整的消息体并反序列化 ---------- */
        byte[] body = new byte[protocol.length];
        in.readBytes(body); // readBytes 会推进 readerIndex，天然"吃掉"这一帧，
                            // 缓冲区中若还剩下一整个完整消息即为粘包场景，下次循环继续解码
        protocol.body = body;
        return protocol;
    }

    /**
     * 用当前帧的 serializeType 反序列化出业务对象
     */
    public <T> T deserializeBody(Class<T> clazz, SerializerFactory factory) {
        Serializer serializer = factory.getSerializer(serializeType);
        return serializer.deserialize(body, clazz);
    }

    // ===== getter/setter（body/serializeType 由 encode 内部维护）=====
    public MessageType getMessageType() { return messageType; }
    public void setMessageType(MessageType messageType) { this.messageType = messageType; }
    public long getRequestId() { return requestId; }
    public void setRequestId(long requestId) { this.requestId = requestId; }
    public int getLength() { return length; }
}
