package com.mini.rpc.serialization;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 序列化器工厂：按算法类型字节分发到具体实现
 *
 * 协议头中的 serializeType 字节 -> 具体序列化器实例。
 * 解码端拿到协议帧后凭这一个字节即可选择正确的反序列化方式，
 * 实现"多算法共存、按需升级"。
 */
public class SerializerFactory {

    /** 算法类型常量：写入协议头 1 字节 */
    public static final byte TYPE_JSON = 0x01;
    // 扩展点：
    // public static final byte TYPE_KRYO = 0x02;
    // public static final byte TYPE_HESSIAN = 0x03;

    private static final Map<Byte, Serializer> SERIALIZERS = new ConcurrentHashMap<>();

    static {
        register(new JsonSerializer());
    }

    /**
     * 注册新算法（可插拔扩展入口）
     */
    public static void register(Serializer serializer) {
        SERIALIZERS.put(serializer.type(), serializer);
    }

    /**
     * 按类型获取（解码侧热路径，每帧都会调用）
     */
    public Serializer getSerializer(byte type) {
        Serializer serializer = SERIALIZERS.get(type);
        if (serializer == null) {
            throw new IllegalArgumentException("不支持的序列化类型: " + type);
        }
        return serializer;
    }

    /**
     * 默认序列化器（编码侧使用）
     */
    public Serializer getDefault() {
        return getSerializer(TYPE_JSON);
    }
}
