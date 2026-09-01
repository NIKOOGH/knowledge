package com.mini.rpc.protocol;

/**
 * 协议常量与魔数定义
 *
 * 为什么需要魔数（Magic Number）？
 * 1. TCP 是字节流，服务端可能收到任意数据（甚至恶意流量、HTTP请求）
 * 2. 魔数是协议的"指纹"：读到非法魔数立即断开连接，避免解码器状态错乱
 */
public final class ProtocolConstants {

    private ProtocolConstants() {
    }

    /**
     * 魔数：本框架自定义的协议指纹 "MRPC" 的 ASCII 值
     * 占 4 个字节：0x4D('M') 0x52('R') 0x50('P') 0x43('C')
     */
    public static final int MAGIC_NUMBER = 0x4D525043;

    /**
     * 当前协议版本号（升级协议时通过版本号做兼容判断）
     */
    public static final byte VERSION = 0x01;

    /**
     * 消息头总长度：
     * Magic(4) + Version(1) + MsgType(1) + SerializeType(1) + RequestId(8) + Length(4)
     * = 19 字节
     */
    public static final int HEADER_LENGTH = 4 + 1 + 1 + 1 + 8 + 4;

    /**
     * 消息体最大长度限制（防止单帧过大耗尽内存 / OOM 攻击）
     */
    public static final int MAX_BODY_LENGTH = 16 * 1024 * 1024; // 16MB
}
