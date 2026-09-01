package com.mini.rpc.core;

import lombok.Data;

import java.io.Serializable;

/**
 * RPC 请求体（协议帧的 body 反序列化结果）
 *
 * 一次远程调用在网络上的"语义载荷"。
 * 服务端依据 interfaceName + methodName + parameterTypes 定位唯一方法，
 * 与 Java 反射规则对齐（重载场景必须带参数类型）。
 */
@Data
public class RpcRequest implements Serializable {

    /** 全局唯一请求ID（雪花/自增均可），响应回传用于配对 */
    private long requestId;

    /**
     * 目标服务全限定名：com.mini.rpc.example.HelloService
     * 注册中心即以此作为服务键
     */
    private String interfaceName;

    /** 方法名 */
    private String methodName;

    /** 参数类型列表（解决方法重载的定位问题） */
    private Class<?>[] parameterTypes;

    /** 实参 */
    private Object[] parameters;
}
