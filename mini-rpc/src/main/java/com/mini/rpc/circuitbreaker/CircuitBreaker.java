package com.mini.rpc.circuitbreaker;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 简易熔断器（对标 Sentinel 的断路器思想）
 *
 * 三态状态机：
 *
 *   失败率>=阈值(且样本数够)
 *  ┌───────────────────────┐
 *  │                       ▼
 * CLOSED                OPEN  ──冷却时间到──▶ HALF_OPEN
 *  ▲                     │                      │
 *  └────探测成功─────────────────────────────────┘
 *                        └────探测失败──────▶ (回到OPEN重新计时)
 *
 * 工作方式（调用侧配合）：
 * if (!breaker.tryAcquire()) return fallback();   // 拒绝=降级
 * try { invoke(); breaker.onSuccess(); }
 * catch (e) { breaker.onError(); throw e; }
 */
public class CircuitBreaker {

    // ==================== 可配置参数 ====================

    /** 触发熔断的最小样本数：太小的失败率没有统计意义（如1/1=100%） */
    private final int minRequestThreshold;
    /** 失败率阈值(0-100)：超过则熔断 */
    private final int failureRateThreshold;
    /** OPEN 态持续时间(ms)，到期转 HALF_OPEN 放探测流量 */
    private final long openStateDurationMs;

    /**
     * 滑动窗口：环形数组统计最近 windowSize 次调用的成败
     * 为什么用环形数组而不是计数器+定时清零？
     * 计数器方案在窗口边界有"突变"问题；环形数组天然平滑，O(1)更新。
     */
    private static final int WINDOW_SIZE = 20;
    private final boolean[] window = new boolean[WINDOW_SIZE]; // true=成功
    private final AtomicInteger windowIndex = new AtomicInteger(0);
    /** 窗口内已积累的调用次数上限值 WINDOW_SIZE（用于minRequestThreshold比较） */
    private final AtomicInteger filledCount = new AtomicInteger(0);

    // ==================== 状态字段 ====================

    enum State { CLOSED, OPEN, HALF_OPEN }
    private volatile State state = State.CLOSED;
    /** 转入 OPEN 的时间戳：用于冷却期判断 */
    private final AtomicLong openSince = new AtomicLong(0);

    public CircuitBreaker(int minRequestThreshold,
                          int failureRateThreshold,
                          long openStateDurationMs) {
        this.minRequestThreshold = minRequestThreshold;
        this.failureRateThreshold = failureRateThreshold;
        this.openStateDurationMs = openStateDurationMs;
    }

    /**
     * 调用前的准入检查（对应 Sentinel 的 entry/tryAcquire）
     *
     * @return true 放行真实调用；false 快速失败 -> 执行降级逻辑
     */
    public synchronized boolean tryAcquire() {
        switch (state) {
            case CLOSED:
            case HALF_OPEN:
                // 两种状态均放行：
                // - HALF_OPEN 只放探测请求（并发场景应由这里保证单次探测，
                //   简化版允许多个探测，靠后的成功/失败结果会收敛状态）
                return true;

            case OPEN:
            default:
                // 冷却期已过？ => 转 HALF_OPEN 并放行一次试探
                long elapsed = System.currentTimeMillis() - openSince.get();
                if (elapsed >= openStateDurationMs) {
                    state = State.HALF_OPEN;
                    return true; // 半开放行探测
                }
                return false;    // 仍在冷却期：直接拒绝，保护下游
        }
    }

    /**
     * 调用成功时回调
     */
    public void onSuccess() {
        recordWindow(true);
        switch (state) {
            case HALF_OPEN:
                // 探测成功 -> 服务恢复，闭环回 CLOSED
                resetToClosed();
                break;
            default:
                break;
        }
    }

    /**
     * 调用失败时回调
     */
    public void onError() {
        recordWindow(false);
        switch (state) {
            case HALF_OPEN:
                // 探测失败 -> 立即重新熔断并重置冷却计时
                trip();
                break;
            case CLOSED:
                // 样本足够 且 失败率超限 -> 触发熔断
                if (filledCount.get() >= Math.min(minRequestThreshold, WINDOW_SIZE)
                        && currentFailureRate() >= failureRateThreshold) {
                    trip();
                }
                break;
            default:
                break;
        }
    }

    // ==================== 内部方法 ====================

    /** 写入滑动窗口 */
    private void recordWindow(boolean success) {
        int idx = Math.floorMod(windowIndex.getAndIncrement(), WINDOW_SIZE);
        window[idx] = success;
        filledCount.updateAndGet(c -> Math.min(c + 1, WINDOW_SIZE));
    }

    /** 当前窗口失败率(%) */
    private int currentFailureRate() {
        int fails = 0;
        for (boolean s : window) {
            if (!s) fails++;
        }
        return fails * 100 / WINDOW_SIZE;
    }

    /** 触发熔断进入 OPEN */
    private void trip() {
        state = State.OPEN;
        openSince.set(System.currentTimeMillis());
    }

    /** 完全恢复：清空窗口回到初始态 */
    private void resetToClosed() {
        java.util.Arrays.fill(window, true);
        filledCount.set(WINDOW_SIZE);
        state = State.CLOSED;
    }

    /** 当前是否处于打开状态（观测用） */
    public boolean isOpen() { return state == State.OPEN; }
    public String getState() { return state.name(); }
}
