package com.mini.rpc.serialization;

/**
 * 序列化器抽象接口
 *
 * RPC 性能关键点之一：网络传输的是字节，任何对象跨进程都必须序列化。
 * 常见实现对比：
 * - JSON(Jackson): 可读性好、通用，但体积大、性能一般
 * - Hessian2:      跨语言、体积小，Dubbo 默认
 * - Kryo:          极快、极小，但不跨语言且需处理循环引用
 * - Protobuf:      体积小性能高、强 Schema，gRPC 默认（需 IDL）
 *
 * 框架通过 type() 字节标识算法，协议头携带类型实现"可插拔多算法"
 */
public interface Serializer {

    /**
     * 序列化：Java 对象 -> 字节数组
     */
    byte[] serialize(Object obj);

    /**
     * 反序列化：字节数组 -> Java 对象
     */
    <T> T deserialize(byte[] bytes, Class<T> clazz);

    /**
     * 序列化算法唯一标识（写入协议头 serializeType 字段）
     */
    byte type();
}
