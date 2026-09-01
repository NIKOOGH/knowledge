package com.mini.rpc.example;

import com.mini.rpc.core.ServiceHandler;
import com.mini.rpc.registry.NacosServiceRegistry;
import com.mini.rpc.transport.server.NettyRpcServer;

import java.util.Scanner;

/**
 * 服务提供者启动示例
 *
 * 启动顺序（标准RPC流程）：
 * 1. Netty 服务端监听端口
 * 2. 发布服务到本地映射表
 * 3. 注册到 Nacos —— 此后消费者即可发现并调用
 */
public class ProviderDemo {

    public static void main(String[] args) throws Exception {
        int port = 9000;
        String nacosAddr = "127.0.0.1:8848";

        // 1. 启动 Netty 服务端（异步，不阻塞主线程）
        NettyRpcServer server = new NettyRpcServer();
        server.startAsync(port);

        // 2. 发布服务：接口名 -> 实现实例
        ServiceHandler.publish(HelloService.class.getName(), new HelloServiceImpl());

        // 3. 注册到 Nacos
        NacosServiceRegistry registry = new NacosServiceRegistry(nacosAddr);
        registry.register(HelloService.class.getName(), "127.0.0.1", port);

        System.out.println(">>> 服务已上线 (端口=" + port + ")，输入回车下线退出");
        // 阻塞主线程保活；回车触发优雅下线：先注销注册中心再关闭Netty，
        // 让Nacos推送摘除 -> 存量请求处理完 -> 关闭。生产必须如此，避免流量直接打到死节点。
        new Scanner(System.in).nextLine();
        registry.deregister(HelloService.class.getName(), "127.0.0.1", port);
        server.shutdown();
    }
}
