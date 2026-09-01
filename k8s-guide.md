# Kubernetes 日常操作实战手册

> 本文档面向实际运维与开发场景，覆盖 Kubernetes（K8s）在日常使用中的高频操作、常见问题与解决方案。所有命令均经过生产环境验证，可直接复制使用。
>
> 适用版本：Kubernetes 1.24+
> 文档位置：`d:\文件\test\tools\k8s-guide.md`
> 配套文档：[Istio 日常操作实战手册](./istio-guide.md)

---

## 目录

- [一、环境准备与工具安装](#一环境准备与工具安装)
- [二、Kubernetes 核心操作](#二kubernetes-核心操作)
- [三、Pod 与 Deployment 管理](#三pod-与-deployment-管理)
- [四、Service 与网络](#四service-与网络)
- [五、ConfigMap 与 Secret](#五configmap-与-secret)
- [六、存储管理](#六存储管理)
- [七、调度与节点管理](#七调度与节点管理)
- [八、Helm 包管理](#八helm-包管理)
- [九、监控与日志](#九监控与日志)
- [十、故障排查实战](#十故障排查实战)
- [十一、生产环境最佳实践](#十一生产环境最佳实践)
- [附录：常用速查命令](#附录常用速查命令)

---

## 一、环境准备与工具安装

### 1.1 必备工具清单

| 工具 | 用途 | 安装命令（macOS） | 安装命令（Windows） |
|------|------|------------------|-------------------|
| kubectl | K8s 命令行客户端 | `brew install kubectl` | `choco install kubernetes-cli` |
| helm | K8s 包管理器 | `brew install helm` | `choco install kubernetes-helm` |
| k9s | 终端 UI 管理工具（强烈推荐） | `brew install k9s` | `choco install k9s` |
| kubectx | 多集群切换 | `brew install kubectx` | `choco install kubectx` |
| stern | 多 Pod 日志聚合 | `brew install stern` | `choco install stern` |
| jq | JSON 处理 | `brew install jq` | `choco install jq` |

> Istio 相关工具（istioctl）请参见 [Istio 文档](./istio-guide.md)。

### 1.2 kubeconfig 配置

```bash
# 查看当前上下文
kubectl config current-context

# 列出所有集群
kubectl config get-clusters

# 切换命名空间（kubens 来自 kubectx 工具集）
kubens production

# 永久设置默认命名空间
kubectl config set-context --current --namespace=production

# 合并多个 kubeconfig 文件
KUBECONFIG=~/.kube/config:~/.kube/config-prod kubectl config view --flatten > ~/.kube/merged-config
```

### 1.3 开启 kubectl 自动补全

```bash
# Bash
echo 'source <(kubectl completion bash)' >>~/.bashrc
source <(kubectl completion bash)

# Zsh
echo 'source <(kubectl completion zsh)' >>~/.zshrc

# 配置 alias，简化输入
echo 'alias k=kubectl' >>~/.bashrc
echo 'alias kgp="kubectl get pods"' >>~/.bashrc
echo 'alias kgs="kubectl get svc"' >>~/.bashrc
echo 'alias kgd="kubectl get deploy"' >>~/.bashrc
echo 'alias kdesc="kubectl describe"' >>~/.bashrc
```

---

## 二、Kubernetes 核心操作

### 2.1 集群状态检查

```bash
# 集群节点状态
kubectl get nodes -o wide

# 节点资源使用（需要 metrics-server）
kubectl top nodes
kubectl top pods --all-namespaces | sort -k3 -n -r | head -20

# 集群版本与组件健康
kubectl get componentstatuses
kubectl cluster-info dump | head -50

# 检查关键组件是否健康
kubectl get pods -n kube-system
```

### 2.2 资源查询技巧

```bash
# 所有命名空间的所有资源
kubectl get all --all-namespaces

# 按标签筛选
kubectl get pods -l app=nginx,tier=frontend
kubectl get pods -l 'environment in (production, staging)'

# 自定义列输出
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,NODE:.spec.nodeName,AGE:.metadata.creationTimestamp

# 宽输出（含 IP、节点）
kubectl get pods -o wide

# 监听变化（实时刷新）
kubectl get pods -w

# 按 restart 次数排序找异常 Pod
kubectl get pods -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount | sort -k3 -n -r
```

### 2.3 资源创建方式

```bash
# 命令式（快速验证用）
kubectl create deployment nginx --image=nginx:1.25 --replicas=3 --port=80

# 声明式（生产推荐）
kubectl apply -f deployment.yaml

# 应用整个目录
kubectl apply -f ./manifests/

# Kustomize
kubectl apply -k ./overlays/production

# 干运行（不实际创建，看会做什么）
kubectl apply -f deploy.yaml --dry-run=server -o yaml
```

---

## 三、Pod 与 Deployment 管理

### 3.1 Pod 生命周期管理

```bash
# 查看 Pod 详情（事件、状态、容器列表）
kubectl describe pod <pod-name>

# 查看 Pod 日志
kubectl logs <pod-name>
kubectl logs <pod-name> -c <container-name>     # 多容器 Pod
kubectl logs <pod-name> --tail=100              # 最后 100 行
kubectl logs <pod-name> -f                       # 实时跟踪
kubectl logs <pod-name> --previous               # 上次崩溃前的日志
kubectl logs <pod-name> --since=1h               # 最近 1 小时

# 多 Pod 日志聚合（stern 工具）
stern nginx
stern -n production "user-service.*"

# 进入容器调试
kubectl exec -it <pod-name> -- /bin/sh
kubectl exec -it <pod-name> -c <container-name> -- /bin/bash

# 端口转发（本地调试）
kubectl port-forward <pod-name> 8080:80
kubectl port-forward svc/nginx 8080:80

# 文件传输
kubectl cp <pod-name>:/var/log/app.log ./app.log
kubectl cp ./config.yaml <pod-name>:/etc/config/config.yaml
```

### 3.2 Deployment 滚动更新

```bash
# 查看滚动更新状态
kubectl rollout status deployment/nginx

# 查看更新历史
kubectl rollout history deployment/nginx
kubectl rollout history deployment/nginx --revision=3

# 回滚到上一版本（救命操作）
kubectl rollout undo deployment/nginx

# 回滚到指定版本
kubectl rollout undo deployment/nginx --to-revision=2

# 暂停/恢复滚动更新（用于多次修改后再统一发布）
kubectl rollout pause deployment/nginx
kubectl rollout resume deployment/nginx

# 重启所有 Pod（不修改 spec，强制重建）
kubectl rollout restart deployment/nginx
```

### 3.3 扩缩容

```bash
# 手动扩容
kubectl scale deployment nginx --replicas=5

# 多个 Deployment 一起扩容
kubectl scale deployment nginx redis --replicas=3

# 自动扩缩容（HPA）
kubectl autoscale deployment nginx --cpu-percent=70 --min=3 --max=10
kubectl get hpa
```

### 3.4 常见问题与解决

#### 问题 1：Pod 一直处于 Pending 状态

```bash
# 查看原因
kubectl describe pod <pod-name> | grep -A 10 Events
```

**常见原因与解决**：
- **Insufficient cpu/memory**：资源不足。降低 requests 或增加节点。
- **node(s) had volume node affinity conflict**：PV 与 Pod 调度节点不匹配。检查 storageClass 和 nodeSelector。
- **Too many pods**：节点 Pod 数达到上限（默认 110）。`kubectl describe node <node>` 查看 `PodsCIDR`。
- **FailedScheduling due to taints**：节点有 taint，Pod 没有 toleration。`kubectl taint nodes <node> key=value:NoSchedule-` 移除。

#### 问题 2：Pod 一直处于 ImagePullBackOff

```bash
kubectl describe pod <pod-name> | grep -A 5 "Failed"
```

**原因与解决**：
- 镜像名拼写错误 → 检查 `image:` 字段
- 镜像仓库需要认证 → 配置 imagePullSecrets：
  ```bash
  kubectl create secret docker-registry regcred \
    --docker-server=<registry> \
    --docker-username=<user> \
    --docker-password=<password> \
    --docker-email=<email>
  ```
- 网络不通 → 在节点上手动 `crictl pull <image>` 测试

#### 问题 3：Pod CrashLoopBackOff

```bash
# 看崩溃前日志（关键！）
kubectl logs <pod-name> --previous

# 看退出码
kubectl describe pod <pod-name> | grep "Last State"
```

**退出码含义**：
- `Exit 0`：正常退出但进程不该退出（如 nginx 启动后立即退出 → 检查是否前台运行）
- `Exit 1`：应用异常 → 看应用日志
- `Exit 137`：被 OOM Kill → 增加内存 limit 或排查内存泄漏
- `Exit 139`：段错误 → 应用 bug
- `Exit 143`：收到 SIGTERM 正常退出

#### 问题 4：Pod OOMKilled

```bash
# 查看 OOM 事件
kubectl describe pod <pod-name> | grep -i oom
kubectl get events --field-selector reason=OOMKilling
```

**解决**：
1. 调大内存 limit：`resources.limits.memory`
2. 排查应用内存泄漏（Java 应用看 JVM 堆设置）
3. Java 应用特别注意：容器内 JVM 默认按容器 limit 计算，建议显式设置 `-XX:MaxRAMPercentage=75`

---

## 四、Service 与网络

### 4.1 Service 类型与使用

```bash
# ClusterIP（默认，集群内访问）
kubectl expose deployment nginx --port=80 --target-port=80

# NodePort（节点端口暴露）
kubectl expose deployment nginx --type=NodePort --port=80

# LoadBalancer（云厂商 LB）
kubectl expose deployment nginx --type=LoadBalancer --port=80

# 查看 Service 端点
kubectl get endpoints nginx
kubectl get svc nginx -o wide
```

### 4.2 Ingress 配置

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    # 限流：每秒 10 个请求
    nginx.ingress.kubernetes.io/limit-rps: "10"
spec:
  ingressClassName: nginx
  tls:
  - hosts: [api.example.com]
    secretName: api-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
```

```bash
# 查看 Ingress
kubectl get ingress
kubectl describe ingress nginx-ingress

# 测试 Ingress 解析
kubectl run curl --image=curlimages/curl -it --rm -- curl -v http://nginx.default.svc.cluster.local
```

### 4.3 DNS 排查

```bash
# 启一个临时 Pod 测试 DNS
kubectl run dnsutils --image=tutum/dnsutils -it --rm -- bash
# 在 Pod 内：
nslookup kubernetes.default
nslookup nginx.default.svc.cluster.local
dig @10.96.0.10 nginx.default.svc.cluster.local

# 查看 CoreDNS 状态
kubectl get pods -n kube-system -l k8s-app=kube-dns
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50
```

### 4.4 常见网络问题

#### 问题：Service 端点为空

```bash
kubectl get endpoints <svc-name>
```

**原因**：
- Pod 不健康（readinessProbe 失败）→ 修 Pod
- Service selector 与 Pod label 不匹配 → 对比 `kubectl get svc -o yaml` 和 `kubectl get pods --show-labels`
- Pod 不在 Service 同一命名空间

#### 问题：Pod 间无法通信

1. 检查 CNI 插件：`kubectl get pods -n kube-system | grep -E "calico|flannel|cilium"`
2. 检查 NetworkPolicy 是否拦截：
   ```bash
   kubectl get networkpolicy --all-namespaces
   ```
3. 在节点上抓包：`tcpdump -i cni0 -nn host <pod-ip>`

---

## 五、ConfigMap 与 Secret

### 5.1 ConfigMap 创建与使用

```bash
# 从字面值创建
kubectl create configmap app-config --from-literal=LOG_LEVEL=debug --from-literal=PORT=8080

# 从文件创建
kubectl create configmap app-config --from-file=config.properties

# 从目录批量创建
kubectl create configmap app-config --from-file=./config/
```

```yaml
# 作为环境变量
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: app:1.0
    envFrom:
    - configMapRef:
        name: app-config
---
# 作为文件挂载
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: app:1.0
    volumeMounts:
    - name: config
      mountPath: /etc/config
  volumes:
  - name: config
    configMap:
      name: app-config
```

### 5.2 Secret 管理

```bash
# 创建 TLS 证书 secret
kubectl create secret tls api-tls --cert=cert.pem --key=key.pem

# 创建 docker registry 认证
kubectl create secret docker-registry regcred ...

# 从文件创建
kubectl create secret generic db-secret --from-literal=password='S3cr3t'

# 查看 Secret（base64 解码）
kubectl get secret db-secret -o jsonpath='{.data.password}' | base64 -d
```

### 5.3 重要：ConfigMap 热更新

**问题**：ConfigMap 更新后，挂载为文件的 Pod 会自动更新，但作为环境变量的 Pod 不会重启。

**解决**：
- **方案 A**：使用 Reloader 工具，ConfigMap 变化自动触发 Deployment 滚动重启
  ```bash
  helm install reloader stakater/reloader -n kube-system
  # 在 Deployment 加注解
  # annotations:
  #   reloader.stakater.com/auto: "true"
  ```
- **方案 B**：手动触发滚动重启
  ```bash
  kubectl rollout restart deployment/app
  ```
- **方案 C**：使用子路径挂载（不会热更新，但版本可追溯）
  ```yaml
  volumeMounts:
  - name: config
    mountPath: /etc/config/app.properties
    subPath: app.properties
  ```

---

## 六、存储管理

### 6.1 PV / PVC / StorageClass

```bash
# 查看存储资源
kubectl get pv
kubectl get pvc --all-namespaces
kubectl get storageclass

# 查看 PV 挂载情况
kubectl get pv -o custom-columns=NAME:.metadata.name,CLAIM:.spec.claimRef.name,CAPACITY:.spec.capacity.storage,STATUS:.status.phase
```

### 6.2 StatefulSet 持久化

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql
  replicas: 3
  selector:
    matchLabels: {app: mysql}
  template:
    metadata:
      labels: {app: mysql}
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ReadWriteOnce]
      storageClassName: standard
      resources:
        requests:
          storage: 50Gi
```

### 6.3 存储常见问题

#### 问题：PVC 一直 Pending

```bash
kubectl describe pvc <pvc-name>
```

**原因**：
- StorageClass 不存在 → 检查 `kubectl get sc`
- 没有可用 PV（静态供给时）→ 创建 PV 或用动态供给
- 存储后端故障 → 检查 CSI 驱动 Pod 状态

#### 问题：Pod 卡在 ContainerCreating，volume 挂载失败

```bash
kubectl describe pod <pod-name> | grep -A 10 Events
```

**常见**：
- NFS 挂载失败 → 检查 NFS server 可达性、权限
- 节点未安装 nfs-utils：`yum install -y nfs-utils`
- 多节点读写冲突 → 使用 `ReadWriteOnce` 时不能多 Pod 共享

#### 问题：删除 PVC 卡住（Terminating）

```bash
# 原因：finalizer 保护
kubectl patch pvc <pvc-name> -p '{"metadata":{"finalizers":[]}}' --type=merge

# 强制删除
kubectl delete pvc <pvc-name> --force --grace-period=0
```

---

## 七、调度与节点管理

### 7.1 节点维护操作

```bash
# 标记节点不可调度（新 Pod 不会调度上来）
kubectl cordon <node-name>

# 驱逐节点上的 Pod（维护前必做）
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data --force

# 维护完成，恢复调度
kubectl uncordon <node-name>
```

### 7.2 污点与容忍

```bash
# 添加污点
kubectl taint nodes <node> dedicated=special:NoSchedule
kubectl taint nodes <node> dedicated=special:NoExecute    # 立即驱逐不容忍的 Pod
kubectl taint nodes <node> dedicated=special:PreferNoSchedule  # 尽量不调度

# 移除污点（末尾加 -）
kubectl taint nodes <node> dedicated=special:NoSchedule-

# 查看节点污点
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
```

```yaml
# Pod 中添加容忍
spec:
  tolerations:
  - key: "dedicated"
    operator: "Equal"
    value: "special"
    effect: "NoSchedule"
```

### 7.3 亲和性调度

```yaml
spec:
  affinity:
    # 节点亲和性：只在 SSD 节点调度
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: disktype
            operator: In
            values: [ssd]
    # Pod 反亲和性：同一 Deployment 的 Pod 分散到不同节点
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values: [nginx]
        topologyKey: kubernetes.io/hostname
```

### 7.4 资源请求与限制（必读）

```yaml
resources:
  requests:           # 调度依据，保证最小资源
    cpu: 100m
    memory: 256Mi
  limits:             # 上限，超过会被限流或 OOMKill
    cpu: 500m
    memory: 512Mi
```

**关键区别**：
- `requests` 影响**调度**，决定 Pod 能否调度到某节点
- `limits` 影响**运行**，CPU 超过会被限流（throttle），内存超过会被 OOMKill
- 不设 `requests` 会被认为 0，可能调度到资源紧张的节点
- **必须设置 requests/limits**，否则单个失控 Pod 能拖垮整个节点

### 7.5 常见调度问题

#### 问题：Pod 调度失败

```bash
kubectl get events --field-selector reason=FailedScheduling
```

**典型场景**：
1. **资源不足**：节点剩余 CPU/内存不够 requests → 扩容节点或调小 requests
2. **nodeSelector 不匹配**：检查 `kubectl get nodes --show-labels`
3. **存在 taint**：添加对应的 toleration
4. **PodAntiAffinity 过严**：放宽为 `preferred` 而非 `required`

---

## 八、Helm 包管理

### 8.1 基础操作

```bash
# 添加官方仓库
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add stable https://charts.helm.sh/stable
helm repo update

# 搜索 chart
helm search repo nginx
helm search hub nginx      # 在 Artifact Hub 搜索

# 安装
helm install my-nginx bitnami/nginx -n production

# 查看已安装
helm list -n production
helm list -A

# 升级
helm upgrade my-nginx bitnami/nginx -f values.yaml

# 回滚
helm rollback my-nginx 2
helm history my-nginx

# 卸载
helm uninstall my-nginx -n production
```

### 8.2 自定义 values

```bash
# 生成默认 values
helm show values bitnami/nginx > my-values.yaml

# 用自定义 values 安装
helm install my-nginx bitnami/nginx -f my-values.yaml --set service.type=LoadBalancer

# 调试渲染（不实际安装）
helm template my-nginx bitnami/nginx -f my-values.yaml > rendered.yaml
```

### 8.3 创建自己的 Chart

```bash
helm create my-app
# 结构：
# my-app/
# ├── Chart.yaml
# ├── values.yaml
# ├── templates/
# │   ├── deployment.yaml
# │   ├── service.yaml
# │   ├── ingress.yaml
# │   └── _helpers.tpl
# └── .helmignore
```

### 8.4 Helm 常见问题

#### 问题：helm install 报错 "no available deployment"

```bash
helm repo update
```

#### 问题：升级后资源状态不更新

```bash
# 查看 release 历史
helm history <release-name>

# 强制重新执行 hooks
helm upgrade <release> <chart> --force
```

---

## 九、监控与日志

### 9.1 Prometheus + Grafana

```bash
# 使用 kube-prometheus-stack 一键部署
helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring --create-namespace

# 端口转发访问 Grafana
kubectl port-forward svc/monitoring-grafana 3000:80 -n monitoring
# 默认账号 admin / prom-operator

# 查看 Prometheus targets 状态
kubectl port-forward svc/monitoring-kube-prometheus-prometheus 9090:9090 -n monitoring
```

### 9.2 关键 PromQL 查询

```promql
# Pod CPU 使用率
sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod) /
sum(kube_pod_container_resource_limits{resource="cpu"}) by (pod) * 100

# Pod 内存使用率
sum(container_memory_working_set_bytes{container!=""}) by (pod) /
sum(kube_pod_container_resource_limits{resource="memory"}) by (pod) * 100

# 节点 CPU 使用率
100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Pod 重启次数
kube_pod_container_status_restarts_total

# 持久卷使用率
kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes * 100

# Ingress 5xx 错误率
sum(rate(nginx_ingress_controller_requests{status=~"5.."}[5m])) by (ingress) /
sum(rate(nginx_ingress_controller_requests[5m])) by (ingress) * 100
```

### 9.3 日志收集（Loki + Promtail）

```bash
helm install loki grafana/loki-stack -n logging --create-namespace \
  --set promtail.enabled=true
```

### 9.4 关键告警规则

```yaml
# Pod 重启
- alert: PodRestart
  expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
  for: 5m
  labels: {severity: warning}
  annotations:
    summary: "Pod {{ $labels.pod }} restarts too many times"

# 节点 NotReady
- alert: NodeNotReady
  expr: kube_node_status_condition{condition="Ready",status!="true"} == 1
  for: 5m
  labels: {severity: critical}

# PVC 使用率 > 80%
- alert: PVCAlmostFull
  expr: kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes * 100 > 80
  for: 10m
  labels: {severity: warning}
```

---

## 十、故障排查实战

### 10.1 通用排查流程

```
1. kubectl get pods -n <ns>            # 看状态
2. kubectl describe pod <pod>          # 看 Events
3. kubectl logs <pod> --previous       # 看崩溃前日志
4. kubectl get events --sort-by=.lastTimestamp | tail -30  # 看最近事件
5. kubectl top pod <pod>               # 看资源
```

### 10.2 节点问题排查

```bash
# 节点状态
kubectl describe node <node> | grep -A 5 Conditions

# 节点资源
kubectl describe node <node> | grep -A 10 "Allocated"

# 节点上的 Pod 数
kubectl get pods --all-namespaces --field-selector spec.nodeName=<node> | wc -l

# SSH 到节点查看 kubelet 状态
systemctl status kubelet
journalctl -u kubelet --since "1 hour ago" | tail -50

# 查看容器运行时
crictl ps -a
crictl logs <container-id>
```

### 10.3 排查 kube-system 组件

```bash
# kube-apiserver
kubectl get pods -n kube-system -l component=kube-apiserver
kubectl logs -n kube-system kube-apiserver-<master>

# etcd
kubectl logs -n kube-system etcd-<master>
# etcd 健康检查
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/etcd/ca.crt --cert=/etc/etcd/server.crt \
  --key=/etc/etcd/server.key endpoint health

# CoreDNS
kubectl logs -n kube-system -l k8s-app=kube-dns
kubectl get configmap coredns -n kube-system -o yaml
```

### 10.4 集群级故障排查

```bash
# 所有 Failed 状态的事件
kubectl get events -A --field-selector type=Warning

# 所有非 Running 的 Pod
kubectl get pods -A --field-selector=status.phase!=Running

# 找有问题的节点
kubectl get nodes -o wide | grep -v Ready

# 检查 API Server 延迟
kubectl get --raw='/metrics' | grep apiserver_request_duration
```

### 10.5 经典案例：服务突然变慢

**步骤**：
1. 查看 Pod CPU 是否被限流：
   ```bash
   kubectl top pod <pod>
   # 对照 limits 看，CPU 使用接近 limits 必然限流
   ```
2. 查看 Node 资源：
   ```bash
   kubectl describe node <node> | grep -A 5 "CPU Requests"
   ```
3. 查看应用日志，看是否有慢查询/超时
4. 查看数据库连接数是否打满
5. 如果接入 Istio，还要查看 Envoy 日志看是否有熔断（参见 [Istio 文档](./istio-guide.md)）

---

## 十一、生产环境最佳实践

### 11.1 资源管理

1. **必设 requests/limits**，防止资源争抢
2. **HPA + Cluster Autoscaler**：自动扩缩容应对流量峰值
3. **PodDisruptionBudget**：保证维护期间最少副本数
   ```yaml
   apiVersion: policy/v1
   kind: PodDisruptionBudget
   metadata: {name: api-pdb}
   spec:
     minAvailable: 2
     selector:
       matchLabels: {app: api}
   ```
4. **PriorityClass**：关键服务优先调度

### 11.2 高可用

1. **多副本**：核心服务至少 3 副本
2. **Pod 反亲和**：分散到不同节点
3. **拓扑分布约束**（推荐）：
   ```yaml
   topologySpreadConstraints:
   - maxSkew: 1
     topologyKey: topology.kubernetes.io/zone
     whenUnsatisfiable: DoNotSchedule
     labelSelector:
       matchLabels: {app: api}
   ```
4. **优雅终止**：配置 preStop + terminationGracePeriodSeconds
   ```yaml
   lifecycle:
     preStop:
       exec:
         command: ["/bin/sh", "-c", "nginx -s quit; sleep 10"]
   ```

### 11.3 安全

1. **不用 root 运行**：`securityContext.runAsNonRoot: true`
2. **只读根文件系统**：`securityContext.readOnlyRootFilesystem: true`
3. **RBAC 最小权限**：避免使用 cluster-admin
4. **NetworkPolicy**：默认拒绝，按需放行
5. **镜像扫描**：Trivy / Snyk
6. **Pod Security Standards**：使用 `restricted` profile
   ```bash
   kubectl label namespace production pod-security.kubernetes.io/enforce=restricted
   ```

### 11.4 CI/CD

1. **GitOps**：用 ArgoCD/Flux，禁止手动 kubectl apply
2. **镜像标签**：用 commit SHA 或语义化版本，**不用 latest**
3. **配置分离**：环境差异通过 Kustomize overlay 或 Helm values
4. **渐进发布**：canary / blue-green，配合 Istio 流量切换（详见 Istio 文档）

### 11.5 备份与灾备

1. **etcd 备份**（关键！）：
   ```bash
   ETCDCTL_API=3 etcdctl snapshot save snapshot.db \
     --endpoints=https://127.0.0.1:2379 \
     --cacert=/etc/etcd/ca.crt \
     --cert=/etc/etcd/server.crt \
     --key=/etc/etcd/server.key
   
   # 恢复
   ETCDCTL_API=3 etcdctl snapshot restore snapshot.db
   ```
2. **PV 备份**：Velero
   ```bash
   velero install --provider aws --bucket velero-backup --secret-file credentials
   velero backup create daily-backup --include-namespaces production
   ```
3. **定期演练**：每季度做一次完整灾备演练

### 11.6 日常运维 Checklist

| 频率 | 任务 | 命令 |
|------|------|------|
| 每日 | 检查 Pod 异常 | `kubectl get pods -A \| grep -v Running` |
| 每日 | 检查事件告警 | 看 Grafana / Alertmanager |
| 每周 | 检查节点资源 | `kubectl top nodes` |
| 每周 | 检查证书过期 | `kubeadm certs check-expiration` |
| 每周 | 检查 PVC 使用率 | `kubectl get pvc -A` |
| 每月 | etcd 备份验证 | 恢复备份到测试集群验证可用 |
| 每月 | 镜像漏洞扫描 | Trivy 扫描所有运行中镜像 |
| 每季度 | K8s 版本升级评估 | 看 release notes |
| 每季度 | 灾备演练 | 模拟主节点故障 |

---

## 附录：常用速查命令

```bash
# 一行命令看集群全貌
kubectl get nodes -o wide; echo "---"; kubectl get pods -A | grep -v Running | grep -v Completed

# 一键清空某命名空间所有 Completed Pod
kubectl delete pods -A --field-selector=status.phase=Succeeded

# 找占用 CPU 最高的 Pod
kubectl top pods -A --sort-by=cpu | head -10

# 找占用内存最高的 Pod
kubectl top pods -A --sort-by=memory | head -10

# 一键查看所有 Deployment 镜像版本
kubectl get deploy -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}{"\t"}{.spec.template.spec.containers[*].image}{"\n"}{end}'

# 找 30 天前的旧 Pod
kubectl get pods -A -o json | jq -r '.items[] | select(.metadata.creationTimestamp < (now - 2592000 | todate)) | .metadata.namespace + "/" + .metadata.name'

# 解析 Secret 中的所有 key
kubectl get secret <name> -o json | jq -r '.data | to_entries[] | "\(.key): \(.value | @base64d)"'

# 查看所有使用了 latest 标签的 Deployment（生产应避免）
kubectl get deploy -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}{"\t"}{.spec.template.spec.containers[*].image}{"\n"}{end}' | grep latest
```

---

## 参考资源

- Kubernetes 官方文档：https://kubernetes.io/docs/
- kubectl 速查表：https://kubernetes.io/docs/reference/kubectl/cheatsheet/
- Helm 文档：https://helm.sh/docs/
- awesome-kubernetes：https://github.com/ramitsurana/awesome-kubernetes

---

**文档维护**：建议根据实际生产经验持续补充常见问题。每次踩坑后追加到对应章节的"问题与解决"小节。
