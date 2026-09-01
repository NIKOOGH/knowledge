package com.mini.rpc.core;

import lombok.Data;

import java.io.Serializable;

/**
 * RPC 响应体
 *
 * 成功时携带 returnValue；业务异常/系统异常统一封装到 exceptionMessage，
 * 由消费端抛出还原 —— 隐藏"远程调用"的细节，让调用方像调本地方法一样 try/catch。
 */
@Data
public class RpcResponse implements Serializable {

    /** 对应请求的 requestId（响应必须回传以配对） */
    private long requestId;

    /** 方法正常执行的返回值 */
    private Object returnValue;

    /** 服务端抛出的异常信息（堆栈字符串，简化处理） */
    private String exceptionMessage;

    /**
     * 快速构造成功响应
     */
    public static RpcResponse success(long requestId, Object value) {
        RpcResponse resp = new RpcResponse();
        resp.setRequestId(requestId);
        resp.setReturnValue(value);
        return resp;
    }

    /**
     * 快速构造失败响应
     */
    public static RpcResponse fail(long requestId, String message) {
        RpcResponse resp = new RpcResponse();
        resp.setRequestId(requestId);
        resp.setExceptionMessage(message);
        return resp;
    }

    /**
     * 是否为错误响应
     */
    public boolean hasException() {
        return exceptionMessage != null;
    }
}
