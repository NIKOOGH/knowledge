package com.mini.rpc.serialization;

import com.fasterxml.jackson.annotation.JsonTypeInfo;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * 基于 Jackson 的 JSON 序列化实现
 *
 * 关键配置：
 * - activateDefaultTyping: 反序列化时保留多态类型信息。
 *   RPC 参数是 Object[]，Jackson 必须知道每个元素的真实类型
 *   才能还原（否则全部变成 LinkedHashMap）
 */
public class JsonSerializer implements Serializer {

    /** Jackson 线程安全，全局单例复用 */
    private static final ObjectMapper MAPPER = new ObjectMapper();

    static {
        // 注册类型信息：序列化输出带 @class 字段，反序列化按该字段还原真实类型
        // NON_FINAL 表示仅对非 final 类型的字段写入类型信息
        MAPPER.activateDefaultTyping(
                MAPPER.getPolymorphicTypeValidator(),
                ObjectMapper.DefaultTyping.NON_FINAL,
                JsonTypeInfo.As.PROPERTY);
        // 遇到未知属性不抛异常（服务端接口演进时向前兼容）
        MAPPER.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    @Override
    public byte[] serialize(Object obj) {
        try {
            return MAPPER.writeValueAsBytes(obj);
        } catch (Exception e) {
            throw new RuntimeException("JSON序列化失败", e);
        }
    }

    @Override
    public <T> T deserialize(byte[] bytes, Class<T> clazz) {
        try {
            return MAPPER.readValue(bytes, clazz);
        } catch (Exception e) {
            throw new RuntimeException("JSON反序列化失败", e);
        }
    }

    @Override
    public byte type() {
        return SerializerFactory.TYPE_JSON;
    }
}
