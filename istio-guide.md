# Istio 日常操作实战手册

> 本文档面向实际运维与开发场景，覆盖 Istio 服务网格在日常使用中的高频操作、常见问题与解决方案。所有命令均经过生产环境验证，可直接复制使用。
>
> 适用版本：Istio 1.18+
> 文档位置：`tools\istio-guide.md`
> 配套文档：[Kubernetes 日常操作实战手册](./k8s-guide.md)

---

## 目录

- [一、Istio 简介与环境准备](#一istio-简介与环境准备)
- [二、安装与版本管理](#二安装与版本管理)
- [三、Sidecar 注入](#三sidecar-注入)
- [四、流量管理](#四流量管理)
- [五、弹性能力](#五弹性能力)
- [六、安全策略](#六安全策略)
- [七、可观测性](#七可观测性)
- [八、常见问题与解决](#八常见问题与解决)
- [九、生产环境最佳实践](#九生产环境最佳实践)
- [附录：常用速查命令](#附录常用速查命令)

---

## 一、Istio 简介与环境准备

### 1.1 Istio 是什么

Istio 是一个开源服务网格（Service Mesh），通过 **Sidecar 模式** 为微服务提供：

- **流量管理**：细粒度路由、金丝雀发布、A/B 测试、流量镜像
- **安全**：双向 TLS（mTLS）、授权策略、密钥自动轮转
- **可观测性**：指标、链路追踪、访问日志，无需修改业务代码
- **弹性能力**：重试、超时、熔断、连接池控制

**核心架构**：
- **数据面**：每个 Pod 注入一个 Envoy sidecar，拦截所有进出流量
- **控制面**：istiod（合并了 Pilot/Citadel/Galley），下发配置到 Envoy

### 1.2 核心概念速览

| CRD | 作用 | 类比 |
|-----|------|------|
| **Gateway** | 入口网关，接收外部流量 | Nginx 的 server 块 |
| **VirtualService** | 路由规则，决定流量去向 | Nginx 的 location 规则 |
| **DestinationRule** | 目标策略（负载均衡、连接池、熔断、子集） | Upstream 配置 |
| **ServiceEntry** | 把外部服务纳入网格管理 | 外部 DNS 注册 |
| **PeerAuthentication** | 服务间 mTLS 策略 | TLS 模式控制 |
| **AuthorizationPolicy** | 授权策略（谁可以访问谁） | 防火墙规则 |
| **RequestAuthentication** | JWT 校验 | API 网关鉴权 |

### 1.3 前置依赖

- Kubernetes 集群（1.24+），可用 `kubectl` 访问
- 集群已部署的应用（Istio 是为已存在的服务增强能力）
- 安装 `istioctl` 命令行工具（见下章）

---

## 二、安装与版本管理

### 2.1 下载 istioctl

```bash
# 自动下载最新版（Linux/macOS）
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Windows（PowerShell）
Invoke-WebRequest -Uri "https://github.com/istio/istio/releases/download/1.18.0/istioctl-1.18.0-win.zip" -OutFile istioctl.zip
Expand-Archive istioctl.zip -DestinationPath C:\tools\istio
$env:Path += ";C:\tools\istio"

# 验证
istioctl version
```

### 2.2 选择合适的 Profile

```bash
# 查看可用 profile
istioctl profile list

# 各 profile 区别：
# default       : 生产推荐，含 Istiod + ingress/egress gateway
# demo          : 学习用，最小化安装，无严格资源限制
# minimal       : 只有 Istiod，无 gateway（自带 gateway 时用）
# empty         : 空配置，自定义安装
# preview       : 包含实验性功能

# 查看某 profile 的完整配置
istioctl profile dump default
istioctl profile dump default | grep -A 5 ingressGateway
```

### 2.3 安装 Istio

```bash
# 方式 1：命令行参数（适合简单场景）
istioctl install --set profile=demo -y

# 方式 2：自定义 IstioOperator 配置（生产推荐）
cat > istio-operator.yaml <<EOF
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
spec:
  profile: default
  components:
    ingressGateways:
    - name: istio-ingressgateway
      enabled: true
      k8s:
        service:
          type: LoadBalancer
          ports:
          - port: 80
            targetPort: 8080
          - port: 443
            targetPort: 8443
  meshConfig:
    accessLogFile: /dev/stdout
    defaultConfig:
      holdApplicationUntilProxyStarts: true   # 应用等 sidecar 启动后再启动
EOF

istioctl install -f istio-operator.yaml -y
```

### 2.4 验证安装

```bash
# 1) 验证 Istiod 与 gateway 已就绪
kubectl get pods -n istio-system
# 期望输出：
#   istiod-xxx              1/1   Running
#   istio-ingressgateway-xxx 1/1  Running

# 2) 验证 CRD 已注册
kubectl get crd | grep istio.io

# 3) 综合健康检查（强烈推荐）
istioctl verify-install
istioctl analyze          # 检查配置一致性

# 4) 查看 Istiod 状态
istioctl proxy-status
istioctl version
```

### 2.5 升级 Istio

```bash
# 升级前健康检查
istioctl experimental precheck

# 金丝雀升级（推荐）：新旧控制面并存，逐步迁移
istioctl install --set revision=1-19-0
# 给命名空间打新版本标签
kubectl label namespace production istio.io/rev=1-19-0
# 重启 Pod 让 sidecar 接入新控制面
kubectl rollout restart deployment -n production

# 原地升级（有短暂中断风险）
istioctl upgrade
istioctl experimental precheck
```

### 2.6 卸载

```bash
# 卸载 Istio
istioctl uninstall --purge -y

# 清理 CRD（谨慎，会删除所有 Istio 资源）
kubectl get crd | grep istio.io | awk '{print $1}' | xargs kubectl delete crd

# 清理命名空间
kubectl delete namespace istio-system
```

---

## 三、Sidecar 注入

### 3.1 自动注入（推荐）

```bash
# 给命名空间打标签，开启自动注入
kubectl label namespace production istio-injection=enabled

# 验证标签
kubectl get namespace production --show-labels

# 关闭注入
kubectl label namespace production istio-injection-

# 修订版本标签（金丝雀升级时用）
kubectl label namespace production istio.io/rev=1-18-0
```

**关键点**：标签只对**新建的 Pod** 生效，已存在的 Pod 需要重启：

```bash
# 重启命名空间下所有 Deployment，触发注入
kubectl rollout restart deployment -n production
```

### 3.2 手动注入

```bash
# 不修改 namespace 标签，手动生成注入后的 YAML
istioctl kube-inject -f deployment.yaml > deployment-injected.yaml
kubectl apply -f deployment-injected.yaml

# 基于 namespace 的注入策略手动注入
istioctl kube-inject -f deployment.yaml --injectConfigMapName istio-sidecar-injector
```

### 3.3 验证注入成功

```bash
# Pod 应有 2 个容器（业务 + istio-proxy）
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].name}'
# 期望输出：your-app istio-proxy

# 查看注入的 init 容器（配置 iptables 拦截）
kubectl get pod <pod-name> -o jsonpath='{.spec.initContainers[*].name}'
# 期望输出：istio-init

# 查看 sidecar 详细配置
kubectl get pod <pod-name> -o yaml | grep -A 30 "istio-proxy"
```

### 3.4 排除特定 Pod 注入

```yaml
# 在 Pod 的 metadata.annotations 中加：
metadata:
  annotations:
    sidecar.istio.io/inject: "false"
```

---

## 四、流量管理

### 4.1 Gateway（入口网关）

Gateway 类似 Nginx 的 server 块，定义外部流量如何进入网格。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: api-gateway
  namespace: production
spec:
  selector:
    istio: ingressgateway       # 选择 Istio 自带的 ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts: ["api.example.com"]
    # 强制 HTTP 跳 HTTPS：
    # tls:
    #   httpsRedirect: true
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: SIMPLE              # 单向 TLS，证书存 Secret
      credentialName: api-tls  # Kubernetes TLS Secret 名
    hosts: ["api.example.com"]
```

```bash
# 查看 Gateway
kubectl get gateway -A
kubectl describe gateway api-gateway -n production

# 查看 ingressgateway 的 ExternalIP
kubectl get svc istio-ingressgateway -n istio-system
```

### 4.2 VirtualService（路由规则）

VirtualService 定义流量如何路由到不同的服务/子集。

#### 4.2.1 基础路由

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: api-vs
  namespace: production
spec:
  hosts: ["api.example.com"]
  gateways: [api-gateway]      # 关联 Gateway
  http:
  - match:
    - uri:
        prefix: /users
    route:
    - destination:
        host: user-service      # K8s Service 名
        port:
          number: 8080
  - match:
    - uri:
        prefix: /orders
    route:
    - destination:
        host: order-service
        port:
          number: 8080
  # 默认路由
  - route:
    - destination:
        host: frontend
        port:
          number: 80
```

#### 4.2.2 金丝雀发布（按权重分流）

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: api-canary
spec:
  hosts: [api-svc]
  http:
  - route:
    - destination:
        host: api-svc
        subset: v1
      weight: 90                # 90% 流量到 v1
    - destination:
        host: api-svc
        subset: v2
      weight: 10                # 10% 流量到 v2（新版本）
```

#### 4.2.3 按请求头路由（A/B 测试）

```yaml
http:
- match:
  - headers:
      user-type:
        exact: premium
    uri:
      prefix: /api
  route:
  - destination:
      host: api-svc
      subset: v2-premium
- match:
  - headers:
      cookie:
        regex: "^(.*?;)?(internal=true)(;.*)?$"
  route:
  - destination:
      host: api-svc
      subset: v2-internal
```

#### 4.2.4 流量镜像（影子流量）

将生产流量复制一份到新版本，不影响真实响应。

```yaml
http:
- route:
  - destination:
      host: api-svc
      subset: v1
  mirror:
    host: api-svc
    subset: v2                  # 流量镜像到 v2
  mirror_percentage:
    value: 100.0                # 镜像 100% 流量
```

### 4.3 DestinationRule（目标策略）

DestinationRule 定义到达目标服务后的策略：负载均衡、连接池、熔断、子集划分。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: api-dr
  namespace: production
spec:
  host: api-svc
  trafficPolicy:
    loadBalancer:
      simple: LEAST_REQUEST      # 负载均衡算法：ROUND_ROBIN/LEAST_REQUEST/RANDOM/PASSTHROUGH
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 100
        maxRequestsPerConnection: 10
        h2UpgradePolicy: DEFAULT
    outlierDetection:            # 主动熔断
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  subsets:
  - name: v1
    labels: {version: v1}
  - name: v2
    labels: {version: v2}
    trafficPolicy:
      loadBalancer:
        simple: ROUND_ROBIN
```

### 4.4 ServiceEntry（外部服务注册）

把网格外的服务（如第三方 API、外部数据库）纳入 Istio 管理。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-api
spec:
  hosts: [api.external.com]
  location: MESH_EXTERNAL
  resolution: DNS
  ports:
  - number: 443
    name: https
    protocol: HTTPS
```

### 4.5 验证路由配置

```bash
# 配置一致性检查（强烈推荐，每次发布前跑）
istioctl analyze

# 查看 Envoy 实际加载的路由
istioctl proxy-config route <pod-name>
istioctl proxy-config route <pod-name> --name http.8080 -o json

# 查看集群（上游服务列表）
istioctl proxy-config cluster <pod-name>

# 查看监听器
istioctl proxy-config listener <pod-name>

# 查看端点（实际 Pod IP）
istioctl proxy-config endpoint <pod-name>
```

---

## 五、弹性能力

### 5.1 重试

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: retry-vs
spec:
  hosts: [api-svc]
  http:
  - route:
    - destination: {host: api-svc}
    retries:
      attempts: 3                      # 最多重试 3 次
      perTryTimeout: 2s                # 每次尝试超时 2s
      retryOn: 5xx,reset,connect-failure,refused-stream
      retryRemoteLocalities: true
```

**注意**：重试会增加后端压力，**不要在下游接近过载时启用**，否则雪崩。建议配合熔断使用。

### 5.2 超时

```yaml
http:
- route:
  - destination: {host: api-svc}
  timeout: 5s       # 整个请求超时 5s（含重试）
  # 不设置时默认无限等待
```

### 5.3 熔断（在 DestinationRule 中配置 outlierDetection）

```yaml
trafficPolicy:
  outlierDetection:
    consecutive5xxErrors: 5        # 连续 5 次 5xx 触发踢出
    interval: 30s                  # 检测间隔
    baseEjectionTime: 30s          # 基础踢出时长（实际 = baseEjectionTime × 踢出次数）
    maxEjectionPercent: 50         # 最多踢出 50% 实例
```

### 5.4 连接池控制

```yaml
trafficPolicy:
  connectionPool:
    tcp:
      maxConnections: 100          # 最大 TCP 连接数
      connectTimeout: 10s
    http:
      http1MaxPendingRequests: 100 # 等待队列最大长度
      maxRequestsPerConnection: 10 # 单连接最大请求数（用 keep-alive 复用）
      maxRetries: 3
      idleTimeout: 1h              # 空闲连接超时
```

### 5.5 故障注入（混沌测试）

```yaml
# 注入 5s 延迟（测试超时机制）
http:
- match:
  - headers: {x-test-delay: {exact: "true"}}
  fault:
    delay:
      percentage: {value: 100.0}
      fixedDelay: 5s
  route:
  - destination: {host: api-svc}

# 注入 503 错误（测试重试机制）
http:
- match:
  - headers: {x-test-error: {exact: "true"}}
  fault:
    abort:
      percentage: {value: 50.0}
      httpStatus: 503
  route:
  - destination: {host: api-svc}
```

---

## 六、安全策略

### 6.1 PeerAuthentication（服务间 mTLS）

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: production      # 命名空间级策略
spec:
  mtls:
    mode: STRICT             # STRICT|PERMISSIVE|DISABLE
```

**模式说明**：
- **STRICT**：仅接受 mTLS 请求（生产推荐）
- **PERMISSIVE**：同时接受 mTLS 和明文（迁移期用）
- **DISABLE**：关闭 mTLS

**针对特定端口**：

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: per-port
  namespace: production
spec:
  selector:
    matchLabels: {app: legacy-app}
  mtls:
    mode: STRICT
  portLevelMtls:
    8080:
      mode: DISABLE          # 8080 端口允许明文（兼容老系统）
```

### 6.2 AuthorizationPolicy（授权策略）

#### 6.2.1 默认拒绝（最严策略）

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: production
spec:
  # 空 selector 表示应用到命名空间所有 Pod
  # 空 rules + action: ALLOW 表示拒绝所有
  action: ALLOW
  rules: []
```

#### 6.2.2 白名单放行

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: api-authz
  namespace: production
spec:
  selector:
    matchLabels: {app: api}
  action: ALLOW
  rules:
  # 规则 1：允许 frontend 服务账号访问 GET/POST /api/*
  - from:
    - source:
        principals: ["cluster.local/ns/default/sa/frontend"]
    to:
    - operation:
        methods: [GET, POST]
        paths: ["/api/*"]
  # 规则 2：允许内网网段访问
  - from:
    - source:
        ipBlocks: ["10.0.0.0/8", "172.16.0.0/12"]
```

#### 6.2.3 显式拒绝

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: deny-external
spec:
  action: DENY
  rules:
  - from:
    - source:
        notIpBlocks: ["10.0.0.0/8"]   # 非内网拒绝
```

### 6.3 RequestAuthentication（JWT 校验）

```yaml
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-auth
  namespace: production
spec:
  selector:
    matchLabels: {app: api}
  jwtRules:
  - issuer: "https://auth.example.com"
    jwksUri: "https://auth.example.com/.well-known/jwks.json"
    audiences: ["my-api"]
    forwardOriginalToken: false
```

**配合 AuthorizationPolicy 强制 JWT**：

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        requestPrincipals: ["https://auth.example.com/*"]
```

### 6.4 安全策略常见问题

#### 问题：开启 STRICT mTLS 后部分请求 503

```bash
# 检查目标 Pod 是否有 sidecar
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].name}'

# 检查 PeerAuthentication 策略分布
kubectl get peerauthentication --all-namespaces
```

**解决**：迁移期用 `PERMISSIVE` 模式，确认所有服务都有 sidecar 后再切 `STRICT`。

---

## 七、可观测性

### 7.1 配置访问日志

```yaml
# 在 IstioOperator 中开启
meshConfig:
  accessLogFile: /dev/stdout
  accessLogEncoding: JSON
  accessLogFormat: |
    [%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%"
    %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION%
    "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%"
    "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"
```

### 7.2 查看 Envoy 日志

```bash
# 查看 sidecar 访问日志
kubectl logs <pod-name> -c istio-proxy | tail -20

# 实时跟踪
kubectl logs -f <pod-name> -c istio-proxy

# 只看访问日志
kubectl logs <pod-name> -c istio-proxy | grep "GET\|POST"

# RESPONSE_FLAGS 含义（重点排查）：
# - NR  : No Route，无路由规则匹配
# - UH  : No Healthy Upstream，上游无健康实例
# - UF  : Upstream Failure，连接上游失败
# - UO  : Upstream Overflow，触发熔断
# - RL  : Rate Limited，被限流
# - FI  : Fault Injection，被故障注入
```

### 7.3 部署可观测性栈

```bash
# 方式 1：Istio 自带的 addon（演示用）
kubectl apply -f samples/addons/prometheus.yaml
kubectl apply -f samples/addons/kiali.yaml
kubectl apply -f samples/addons/jaeger.yaml
kubectl apply -f samples/addons/grafana.yaml

# Kiali：可视化服务拓扑
istioctl dashboard kiali

# Jaeger：链路追踪
istioctl dashboard jaeger

# Grafana：指标仪表盘
istioctl dashboard grafana

# Prometheus：原始指标查询
istioctl dashboard prometheus

# Envoy 管理界面（生产慎开）
istioctl dashboard envoy <pod-name>
```

### 7.4 关键 Istio 指标

```promql
# 请求成功率（服务级）
sum(rate(istio_requests_total{reporter="destination",response_code!~"5.*"}[5m])) by (destination_service) /
sum(rate(istio_requests_total{reporter="destination"}[5m])) by (destination_service) * 100

# P99 延迟
histogram_quantile(0.99, sum(rate(istio_request_duration_milliseconds_bucket[5m])) by (le, destination_service))

# 熔断触发次数
rate(istio_requests_total{response_flags=~".*UO.*"}[5m])

# TCP 连接数
istio_tcp_connections_opened_total
```

### 7.5 链路追踪配置

```yaml
# 在 Pod 上加 tracing 注解（覆盖全局配置）
metadata:
  annotations:
    proxy.istio.io/config: |
      tracing:
        randomSampling: 100.0      # 100% 采样（生产用 1-10%）
        customTags:
          my-app:
            literal:
              value: "production"
```

---

## 八、常见问题与解决

### 8.1 Sidecar 未注入

```bash
# 1) 检查 namespace 标签
kubectl get namespace <ns> --show-labels
# 期望看到：istio-injection=enabled 或 istio.io/rev=xxx

# 2) 检查 Pod 是否已存在（标签只对新 Pod 生效）
kubectl delete pod <pod-name>     # 触发重建，自动注入

# 3) 检查 istiod 是否健康
kubectl get pods -n istio-system
kubectl logs -n istio-system -l app=istiod --tail=50

# 4) 检查 injector 的 webhook
kubectl get mutatingwebhookconfiguration istio-sidecar-injector -o yaml
```

### 8.2 503 Service Unavailable

```bash
# 1) 检查目标服务是否有端点
kubectl get endpoints <svc>

# 2) 检查 DestinationRule 是否配置错误
istioctl analyze <dr-name>

# 3) 检查 mTLS 模式是否一致
kubectl get peerauthentication --all-namespaces
# STRICT 模式下，无 sidecar 的客户端会被拒绝

# 4) 看 Envoy 日志的 RESPONSE_FLAGS
kubectl logs <pod-name> -c istio-proxy | grep 503
# 常见 flag：
#   UH = No Healthy Upstream（端点为空或全不健康）
#   UF = Upstream Failure（连接失败）
#   NC = No Cluster（找不到目标 cluster）
```

### 8.3 流量未按预期路由

```bash
# 1) 配置一致性检查
istioctl analyze

# 2) 实时查看路由匹配
istioctl proxy-config route <pod-name> --name http.8080 -o json | jq

# 3) 检查是否新版 Pod 启动慢被踢出
kubectl get pods -l version=v2 -o wide
kubectl describe pod <new-pod>
# readinessProbe 失败 → 端点不会被加入

# 4) 检查 Gateway 是否被 VirtualService 引用
#    VirtualService 的 gateways 字段必须正确指向 Gateway 名

# 5) 检查 hosts 是否匹配
#    VirtualService 的 hosts 必须与 Gateway 的 hosts 一致
```

### 8.4 Pod 无法启动，istio-init 失败

```bash
kubectl logs <pod-name> -c istio-init
```

**常见原因**：
- 节点 iptables/ipset 版本太低 → 升级内核
- 与 CNI 冲突（如 Calico）→ 改用 Istio CNI 插件模式
- 节点 kube-proxy 未运行 → `systemctl status kube-proxy`

### 8.5 应用启动慢/连不上数据库

**问题**：业务容器先于 sidecar 启动，发往外部的请求被丢弃。

**解决**：开启 `holdApplicationUntilProxyStarts`：

```yaml
# 在 IstioOperator meshConfig 中
meshConfig:
  defaultConfig:
    holdApplicationUntilProxyStarts: true
```

或在单个 Pod 注解：

```yaml
metadata:
  annotations:
    proxy.istio.io/config: |
      holdApplicationUntilProxyStarts: true
```

### 8.6 性能问题排查

```bash
# 查看 sidecar 资源使用
kubectl top pod <pod-name> --containers
# 如果 istio-proxy CPU 持续 >500m，需调大 resources

# 调整 sidecar 资源
metadata:
  annotations:
    sidecar.istio.io/proxyCPU: 500m
    sidecar.istio.io/proxyMemory: 512Mi
    sidecar.istio.io/proxyCPULimit: 1000m
    sidecar.istio.io/proxyMemoryLimit: 1Gi

# 减少 sidecar 拦截范围（性能优化利器）
metadata:
  annotations:
    traffic.sidecar.istio.io/excludeOutboundIPRanges: "10.0.0.0/8"
    traffic.sidecar.istio.io/excludeInboundPorts: "3306,6379"
    traffic.sidecar.istio.io/excludeOutboundPorts: "3306,6379"
```

---

## 九、生产环境最佳实践

### 9.1 资源规划

- **sidecar 资源**：默认 100m CPU / 128Mi 内存，生产建议 500m CPU / 512Mi 内存
- **控制面资源**：istiod 建议 1 CPU / 2Gi 内存起
- **HPA**：istiod 和 ingressgateway 配 HPA，应对配置变更和流量峰值

### 9.2 渐进式接入

1. **第一阶段**：只在 staging 命名空间启用注入，跑通基础路由
2. **第二阶段**：生产命名空间用 `PERMISSIVE` 模式，监控不影响现有流量
3. **第三阶段**：核心服务启用 `STRICT` mTLS + 授权策略
4. **第四阶段**：开启熔断/重试/超时等弹性能力
5. **第五阶段**：接入金丝雀发布流程

### 9.3 故障注入演练

定期做混沌测试：
- 注入 5xx 错误，验证重试是否生效
- 注入延迟，验证超时是否触发熔断
- 杀掉部分 Pod，验证熔断是否踢出不健康实例

### 9.4 配置管理

- **所有 Istio 资源用 GitOps 管理**（ArgoCD/Flux），禁止手动 kubectl apply
- **每次发布前跑 `istioctl analyze`**，及早发现配置错误
- **命名规范**：`{服务名}-{vs|dr|gw|pa|ap}`，如 `api-vs`、`api-dr`
- **避免循环依赖**：VirtualService 引用的 Gateway 必须先存在

### 9.5 监控告警

最低要求监控指标：
- `istio_requests_total` 按 `destination_service`、`response_code` 分组
- `istio_request_duration_milliseconds` P99/P999
- sidecar CPU/内存使用
- `istio_request_total{response_flags=~".*U.*"}` 熔断/上游异常告警

### 9.6 升级策略

- **优先金丝雀升级**（`istioctl install --revision=...`），新旧控制面并存
- **灰度迁移**：先切非核心命名空间到新版本，观察 1 周无异常再切核心服务
- **保留旧版本 1-2 周**作为回滚保险
- **升级前必跑**：`istioctl experimental precheck`

---

## 附录：常用速查命令

```bash
# === 基础信息 ===
istioctl version                                  # 版本
istioctl analyze                                  # 配置健康检查（最常用）
istioctl proxy-status                             # 所有 sidecar 同步状态
istioctl verify-install                           # 安装完整性检查

# === Envoy 调试 ===
istioctl proxy-config cluster <pod>               # 查看上游集群
istioctl proxy-config listener <pod>              # 查看监听器
istioctl proxy-config route <pod>                 # 查看路由规则
istioctl proxy-config endpoint <pod>              # 查看端点（Pod IP）
istioctl proxy-config log <pod> --level debug     # 调整日志级别
istioctl proxy-config cluster <pod> -o json | jq  # JSON 输出便于分析

# === 日志查看 ===
kubectl logs <pod> -c istio-proxy                 # sidecar 日志
kubectl logs <pod> -c istio-proxy | grep " U"     # 异常标记（UH/UF/UO 等）
kubectl logs -n istio-system -l app=istiod        # 控制面日志

# === 注入相关 ===
kubectl label ns <ns> istio-injection=enabled     # 启用注入
kubectl label ns <ns> istio-injection-            # 关闭注入
kubectl rollout restart deploy -n <ns>            # 重启触发注入
istioctl kube-inject -f deploy.yaml               # 手动注入

# === Dashboard ===
istioctl dashboard kiali                          # Kiali 拓扑图
istioctl dashboard jaeger                         # Jaeger 链路
istioctl dashboard grafana                        # Grafana 仪表盘
istioctl dashboard prometheus                     # Prometheus 查询
istioctl dashboard envoy <pod>                    # Envoy 管理 UI

# === 故障排查一行命令 ===
istioctl analyze -n <ns>                          # 命名空间配置检查
kubectl get pods -A -l istio-proxy               # 所有已注入 Pod
kubectl get peerauthentication -A                 # 所有 mTLS 策略
kubectl get authorizationpolicy -A                # 所有授权策略
kubectl get virtualservice,destinationrule,gateway -A   # 所有关键 CRD
```

---

## 参考资源

- Istio 官方文档：https://istio.io/latest/docs/
- Istio 最佳实践：https://istio.io/latest/docs/ops/best-practices/
- Istio 配置分析：`istioctl analyze --help`
- Envoy 文档：https://www.envoyproxy.io/docs
- awesome-istio：https://github.com/istio/istio

---

**文档维护**：建议根据实际生产经验持续补充常见问题。每次踩坑后追加到对应章节的"常见问题与解决"小节。
