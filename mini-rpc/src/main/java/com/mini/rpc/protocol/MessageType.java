package com.mini.rpc.protocol;

/**
 * 消息类型枚举
 *
 * RPC 框架中除了请求/响应，通常还需要心跳等控制消息保持长连接活性。
 * Netty 空闲检测 + 心跳是长连接的标准做法（本框架预留 HEARTBEAT）。
 */
public enum MessageType {

    /** 客户端 -> 服务端：RPC 请求 */
    REQUEST((byte) 0x01),

    /** 服务端 -> 客户端：RPC 响应 */
    RESPONSE((byte) 0x02),

    /** 心跳包，防止中间设备（NAT/LB）回收空闲连接 */
    HEARTBEAT((byte) 0x03);

    private final byte code;

    MessageType(byte code) {
        this.code = code;
    }

    public byte getCode() {
        return code;
    }

    /**
     * 编解码时根据字节值反查枚举
     */
    public static MessageType of(byte code) {
        for (MessageType type : values()) {
            if (type.code == code) {
                return type;
            }
        }
        throw new IllegalArgumentException("未知消息类型: " + code);
    }
}
