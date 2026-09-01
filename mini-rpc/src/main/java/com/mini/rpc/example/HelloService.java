package com.mini.rpc.example;

/**
 * 示例服务接口（生产中应放在独立的 api 模块，供 Provider/Consumer 共同依赖）
 */
public interface HelloService {

    /**
     * 打招呼
     *
     * @param name 姓名
     * @return 问候语
     */
    String sayHi(String name);

    /**
     * 两数相加（验证参数与返回值的序列化）
     */
    int add(int a, int b);
}
