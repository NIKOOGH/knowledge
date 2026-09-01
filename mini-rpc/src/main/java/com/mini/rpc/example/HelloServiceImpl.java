package com.mini.rpc.example;

/**
 * 服务实现（Provider侧持有）
 */
public class HelloServiceImpl implements HelloService {

    @Override
    public String sayHi(String name) {
        // 标记当前线程，验证调用确实发生在服务端业务线程池中
        return "你好, " + name + "! (来自 " + Thread.currentThread().getName() + ")";
    }

    @Override
    public int add(int a, int b) {
        return a + b;
    }
}
