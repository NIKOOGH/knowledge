package com.mini.rpc.transport.server;

import com.mini.rpc.protocol.ProtocolConstants;
import com.mini.rpc.protocol.ProtocolDecoder;
import com.mini.rpc.protocol.ProtocolEncoder;
import com.mini.rpc.serialization.SerializerFactory;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.channel.ChannelInitializer;
import io.netty.channel.ChannelOption;
import io.netty.channel.ChannelPipeline;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import lombok.extern.slf4j.Slf4j;

/**
 * Netty RPC 服务端
 *
 * Pipeline 分层（自上而下 = 入站顺序）：
 *   ProtocolDecoder  —— 按帧切包+反序列化头（解决粘包/拆包）
 *   ProtocolEncoder  —— 出站：对象 -> 二进制帧
 *   ServerHandler    —— 业务：反射调用（内部转业务线程池）
 */
@Slf4j
public class NettyRpcServer {

    /** Boss线程组：只负责 accept 新连接 */
    private final EventLoopGroup bossGroup = new NioEventLoopGroup(1);
    /**
     * Worker线程组：处理已建立连接的 IO 读写。
     * 默认 CPU核数*2；本项目追求IO效率最大化，业务的反射调用被
     * ServerHandler 转移到独立线程池，避免阻塞 worker。
     */
    private final EventLoopGroup workerGroup = new NioEventLoopGroup();

    private final SerializerFactory serializerFactory = new SerializerFactory();

    public void start(int port) {
        ServerBootstrap bootstrap = new ServerBootstrap();
        bootstrap.group(bossGroup, workerGroup)
                .channel(NioServerSocketChannel.class)          // NIO 多路复用模式
                .childHandler(new ChannelInitializer<SocketChannel>() {
                    @Override
                    protected void initChannel(SocketChannel ch) {
                        ChannelPipeline p = ch.pipeline();
                        // 协议编解码器（有状态的对象由pipeline为每条连接独立实例化一份，无并发问题）
                        p.addLast(new ProtocolEncoder(serializerFactory));
                        p.addLast(new ProtocolDecoder(serializerFactory));
                        // 业务处理器（内部自带业务线程池切换）
                        p.addLast(new ServerHandler());
                    }
                })
                /*
                 * TCP 层参数调优：
                 * SO_BACKLOG: 半/全连接队列长度，突发建连时的缓冲
                 * TCP_NODELAY: 禁用 Nagle 算法。RPC 小包场景若开启，
                 *              会为"攒够MSS"而延迟发送，徒增延迟
                 * SO_KEEPALIVE: TCP层保活探针（应用层另有心跳双保险）
                 */;
        bootstrap.option(ChannelOption.SO_BACKLOG, 1024);
        bootstrap.childOption(ChannelOption.TCP_NODELAY, true);
        bootstrap.childOption(ChannelOption.SO_KEEPALIVE, true);

        try {
            io.netty.channel.ChannelFuture future = bootstrap.bind(port).sync();
            log.info("Netty RPC 服务端启动成功, 端口={}", port);
            // 阻塞等待关闭命令（示例中常驻进程）
            future.channel().closeFuture().sync();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            log.error("服务端启动被中断", e);
        } finally {
            shutdown();
        }
    }

    public void shutdown() {
        bossGroup.shutdownGracefully();
        workerGroup.shutdownGracefully();
        log.info("服务端已优雅关闭");
    }

    /**
     * 后台启动（不阻塞），供 ProviderDemo 在注册完服务后自行 hold 主线程
     */
    public void startAsync(int port) {
        new Thread(() -> start(port), "rpc-server-main").start();
    }
}
