package com.mini.rpc.transport.client;

import com.mini.rpc.core.RpcRequest;
import com.mini.rpc.core.RpcResponse;
import com.mini.rpc.core.RequestHolder;
import com.mini.rpc.protocol.MessageType;
import com.mini.rpc.protocol.ProtocolDecoder;
import com.mini.rpc.protocol.ProtocolEncoder;
import com.mini.rpc.protocol.RpcProtocol;
import com.mini.rpc.protocol.RpcProtocolHolder;
import com.mini.rpc.serialization.SerializerFactory;
import io.netty.bootstrap.Bootstrap;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFutureListener;
import io.netty.channel.ChannelInitializer;
import io.netty.channel.ChannelOption;
import io.netty.channel.ChannelPipeline;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioSocketChannel;
import lombok.extern.slf4j.Slf4j;

import java.net.InetSocketAddress;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;

/**
 * Netty RPC 客户端（消费端网络传输核心）
 *
 * 职责：
 * 1. 到各服务节点的长连接管理（懒建连 + 缓存复用）
 * 2. 发送请求并异步转同步返回 RpcResponse
 *
 * 长连接复用的意义：TCP 三次握手+TLS 握手成本高，
 * RPC 高频小包必须复用连接；一条 Channel 上并发多请求靠 requestId 多路复用。
 */
@Slf4j
public class NettyRpcClient {

    /** Worker线程组：所有连接共享的 IO 线程 */
    private final EventLoopGroup group = new NioEventLoopGroup();
    private final SerializerFactory serializerFactory = new SerializerFactory();

    /** 连接缓存: "host:port" -> 已建立的Channel（延迟关闭策略略） */
    private final Map<String, Channel> channelCache = new ConcurrentHashMap<>();

    /** 请求ID生成器：本客户端实例内自增即可保证唯一配对 */
    private static final java.util.concurrent.atomic.AtomicLong REQUEST_ID_GEN =
            new java.util.concurrent.atomic.AtomicLong(System.currentTimeMillis());

    /**
     * 同步调用入口：代理层最终会走到这里
     *
     * @param address   目标节点
     * @param request   待发送请求
     * @param timeoutMs 超时时间
     */
    public RpcResponse send(InetSocketAddress address, RpcRequest request, long timeoutMs)
            throws Exception {
        // 1. 申请(或复用)到目标节点的连接
        Channel channel = getOrCreateChannel(address);
        if (channel == null || !channel.isActive()) {
            throw new IllegalStateException("无法连接服务节点: " + address);
        }

        // 2. 创建与该请求绑定的 future，注册后挂起等待
        CompletableFuture<Object> future = new CompletableFuture<>();
        RequestHolder.register(request.getRequestId(), future);

        try {
            // 3. 组协议帧并异步写出
            RpcProtocol protocol = new RpcProtocol();
            protocol.setMessageType(MessageType.REQUEST);
            protocol.setRequestId(request.getRequestId());
            com.mini.rpc.serialization.Serializer serializer = serializerFactory.getDefault();

            // writeAndFlush 立即返回，真正的写出由 EventLoop 异步完成
            channel.writeAndFlush(new RpcProtocolHolder(protocol, request, serializer.type()))
                    .addListener((ChannelFutureListener) f -> {
                        if (!f.isSuccess()) {
                            // 写失败让等待方立即收到异常而非傻等超时
                            RequestHolder.removeAndFail(request.getRequestId(), f.cause());
                        }
                    });

            // 4. 挂起等待响应 / 异常完成 / 超时
            Object result = future.get(timeoutMs, TimeUnit.MILLISECONDS);
            return (RpcResponse) result;
        } catch (java.util.concurrent.TimeoutException e) {
            // 超时要清理pending, 否则Map膨胀且迟到响应会造成内存泄漏
            RequestHolder.removeAndFail(request.getRequestId(),
                    new RuntimeException("RPC调用超时: " + timeoutMs + "ms"));
            throw e;
        } finally {
            // 兜底移除（正常路径complete()已remove，幂等）
        }
    }

    /**
     * 懒建连 + 缓存复用。双检锁防并发重复建连。
     */
    private Channel getOrCreateChannel(InetSocketAddress address) throws InterruptedException {
        String key = address.getHostString() + ":" + address.getPort();
        Channel cached = channelCache.get(key);
        if (cached != null && cached.isActive()) {
            return cached;
        }

        synchronized (this) {
            if (channelCache.get(key) != null && channelCache.get(key).isActive()) {
                return channelCache.get(key); // 双检：别的线程可能刚建好
            }
            Bootstrap bootstrap = new Bootstrap()
                    .group(group)
                    .channel(NioSocketChannel.class)
                    .option(ChannelOption.TCP_NODELAY, true)      // 同服务端：关Nagle
                    .option(ChannelOption.CONNECT_TIMEOUT_MILLIS, 3000)
                    .handler(new ChannelInitializer<SocketChannel>() {
                        @Override
                        protected void initChannel(SocketChannel ch) {
                            ChannelPipeline p = ch.pipeline();
                            p.addLast(new ProtocolEncoder(serializerFactory));
                            p.addLast(new ProtocolDecoder(serializerFactory));
                            p.addLast(new ClientHandler());
                        }
                    });

            Channel channel = bootstrap.connect(address).sync().channel();
            channelCache.put(key, channel);
            log.info("建立新连接 -> {}", key);
            return channel;
        }
    }

    /**
     * 发送前生成唯一requestId（也供代理层使用，静态便于入口统一）
     */
    public static long nextRequestId() {
        return REQUEST_ID_GEN.incrementAndGet();
    }

    /** 关闭全部资源（进程退出时） */
    public void close() {
        channelCache.values().forEach(Channel::close);
        group.shutdownGracefully();
    }
}
