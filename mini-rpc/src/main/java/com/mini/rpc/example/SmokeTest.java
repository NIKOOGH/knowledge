package com.mini.rpc.example;


import com.mini.rpc.core.RpcRequest;
import com.mini.rpc.core.RpcResponse;
import com.mini.rpc.core.ServiceHandler;
import com.mini.rpc.transport.client.NettyRpcClient;
import com.mini.rpc.transport.server.NettyRpcServer;

import java.net.InetSocketAddress;

/**
 * 冒烟测试（临时）：绕过Nacos直连验证 协议编解码+传输+反射调用 全链路
 */
public class SmokeTest {

    public static void main(String[] args) throws Exception {
        int port = 19099;
        ServiceHandler.publish(HelloService.class.getName(), new HelloServiceImpl());
        new NettyRpcServer().startAsync(port);
        Thread.sleep(1500); // 等待服务端就绪

        NettyRpcClient client = new NettyRpcClient();
        for (int i = 0; i < 20; i++) { // 连发20次观察粘包/拆包处理稳定性
            RpcRequest req = new RpcRequest();
            req.setRequestId(NettyRpcClient.nextRequestId());
            req.setInterfaceName(HelloService.class.getName());
            req.setMethodName(i % 2 == 0 ? "sayHi" : "add");
            req.setParameterTypes(i % 2 == 0
                    ? new Class<?>[]{String.class}
                    : new Class<?>[]{int.class, int.class});
            req.setParameters(i % 2 == 0
                    ? new Object[]{"u" + i}
                    : new Object[]{i, i * 10});

            RpcResponse resp = client.send(new InetSocketAddress("127.0.0.1", port), req, 3000);
            System.out.println("结果" + i + ": "
                    + (resp.hasException() ? "ERROR " + resp.getExceptionMessage() : resp.getReturnValue()));
        }
        System.out.println("SMOKE-TEST-PASS");
        System.exit(0);
    }
}
