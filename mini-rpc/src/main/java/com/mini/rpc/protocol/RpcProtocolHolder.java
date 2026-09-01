package com.mini.rpc.protocol;

import lombok.AllArgsConstructor;
import lombok.Getter;

/**
 * 编码器入参持有对象
 *
 * 把"协议骨架"与"业务载荷"一起传给编码器，
 * 由 encode 阶段统一完成序列化并填充 length 字段。
 */
@Getter
@AllArgsConstructor
public class RpcProtocolHolder {

    /** 协议骨架（消息类型/请求ID 已填好） */
    private final RpcProtocol protocol;

    /** 待序列化的业务对象（RpcRequest / RpcResponse） */
    private final Object payload;

    /** 序列化算法类型 */
    private final byte serializeType;
}
