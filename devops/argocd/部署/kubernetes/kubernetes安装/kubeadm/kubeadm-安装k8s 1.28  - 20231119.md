# kubeadm-安装k8s 1.28  - 20231119

## 背景

最近研究K8S的高可用部署方式 ，几种方式都需要手工试试。

## 参考

https://kubernetes.io/zh-cn/docs/setup/production-environment/tools/kubeadm/install-kubeadm

https://www.cpweb.top/1644

https://www.lixueduan.com/posts/kubernetes/01-install/

## 准备开始

* 一台兼容的 Linux 主机。Kubernetes 项目为基于 Debian 和 Red Hat 的 Linux 发行版以及一些不提供包管理器的发行版提供通用的指令。
* 每台机器 2 GB 或更多的 RAM（如果少于这个数字将会影响你应用的运行内存）。
* CPU 2 核心及以上。
* 集群中的所有机器的网络彼此均能相互连接（公网和内网都可以）。
* 节点之中不可以有重复的主机名、MAC 地址或 product_uuid。请参见[这里](https://kubernetes.io/zh-cn/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#verify-mac-address)了解更多详细信息。
* 开启机器上的某些端口。请参见[这里](https://kubernetes.io/zh-cn/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#check-required-ports)了解更多详细信息。
* 交换分区的配置。kubelet 的默认行为是在节点上检测到交换内存时无法启动。 kubelet 自 v1.22 起已开始支持交换分区。自 v1.28 起，仅针对 cgroup v2 支持交换分区； kubelet 的 NodeSwap 特性门控处于 Beta 阶段，但默认被禁用。

  * 如果 kubelet 未被正确配置使用交换分区，则你**必须**禁用交换分区。 例如，`sudo swapoff -a`​ 将暂时禁用交换分区。要使此更改在重启后保持不变，请确保在如 `/etc/fstab`​、`systemd.swap`​ 等配置文件中禁用交换分区，具体取决于你的系统如何配置。

## 操作系统环境

|主机名|ip|配置|角色|
| -----------| ----------------| -----------| --------|
|kubeadm-1|192.168.31.52|2c/8g/50G|master|
|kubeadm-2|192.168.31.108|2c/8g/50G|node|
|kubeadm-3|192.168.31.33|2c/8g/50G|node|

‍

## 基础准备

关闭防火封墙

```shell
systemctl stop firewalld && systemctl disable firewalld
```

关闭swap

```shell
swapoff -a
sed -i 's/^.*centos-swap/#&/g' /etc/fstab
```

主机映射

```shell
cat << EOF >> /etc/hosts
192.168.31.52 kubeadm-1
192.168.31.108 kubeadm-2
192.168.31.33 kubeadm-3
EOF
```

 内核参数

```shell
# 激活 br_netfilter 模块
modprobe br_netfilter
cat << EOF > /etc/modules-load.d/k8s.conf
br_netfilter
EOF

# 内核参数设置：开启IP转发，允许iptables对bridge的数据进行处理
cat << EOF > /etc/sysctl.d/k8s.conf 
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF

# 立即生效
sysctl --system
```

集群时间同步

master节点

```shell
apt-get install -y chrony
sed -i 's/^server/#&/' /etc/chrony.conf
cat >> /etc/chrony.conf << EOF
#server ntp1.aliyun.com iburst
local stratum 10
allow 192.168.31.0/24
EOF
systemctl restart chronyd
systemctl enable chronyd
```

node节点

```shell
apt-get install -y chrony
sed -i 's/^server/#&/' /etc/chrony.conf
cat >> /etc/chrony.conf  << EOF
server 192.168.31.52 iburst
EOF
systemctl restart chronyd
systemctl enable chronyd
```

* container为基础容器，则执行如下操作

  ```shell
  cat >> /etc/modules-load.d/containerd.conf << EOF
  overlay
  br_netfilter
  EOF

  modprobe overlay
  modprobe br_netfilter

  ## 安装container
  apt-get install containerd -y

  ## 生成默认 containerd 配置文件
  mkdir -p /etc/containerd
  containerd config default |  tee /etc/containerd/config.toml
  ## 使用 systemd cgroup 驱动程序
  vim /etc/containerd/config.toml
  # 把配置文件中的 SystemdCgroup 修改为 true
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
    ...
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
      SystemdCgroup = true
  ## 或者一键命令
  sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml

  ## 用国内源替换 containerd 默认的 sand_box 镜像，编辑 /etc/containerd/config.toml
  [plugins]
    .....
    [plugins."io.containerd.grpc.v1.cri"]
    	...
  	sandbox_image = "registry.aliyuncs.com/google_containers/pause:3.5"
  ## 或者一键命令
  # 需要对路径中的/ 进行转移，替换成\/
  sed -i 's/k8s.gcr.io\/pause/registry.aliyuncs.com\/google_containers\/pause/g' /etc/containerd/config.toml
  ## 配置镜像加速器地址
  [plugins."io.containerd.grpc.v1.cri".registry]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
    # 添加下面两个配置
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
        endpoint = ["https://ekxinbbh.mirror.aliyuncs.com"]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
        endpoint = ["https://gcr.k8s.li"]

  ## 启动
  systemctl daemon-reload
  systemctl enable containerd --now

  ```

* docker为基础容器，则执行如下操作

  ```shell
  # 激活 br_netfilter 模块
  modprobe br_netfilter
  cat << EOF > /etc/modules-load.d/k8s.conf
  br_netfilter
  EOF

  apt-get update
  apt-get -y install apt-transport-https ca-certificates curl software-properties-common
  curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo apt-key add -
  add-apt-repository "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable"
  apt-get -y update
  apt install docker 
  ## cri-docker可以在这个git下载
  https://github.com/Mirantis/cri-dockerd/releases/tag/v0.3.7

  ## 配置docker使用systemd 
  vim /etc/docker/daemon.json 
  {
    "exec-opts":["native.cgroupdriver=systemd"],
    ## 添加国内源,这行注意删除
    "registry-mirrors": [
      "https://docker.mirrors.ustc.edu.cn/",
      "https://hub-mirror.c.163.com",
      "https://registry.docker-cn.com",
      "https://bxsfpjcb.mirror.aliyuncs.com",
      "https://registry.cn-hangzhou.aliyuncs.com"
    ]
  }
  ##  调整cri-docker pause镜像使用过阿里源，默认使用的是国外的，下载不了pause
  vi /lib/systemd/system/cri-docker.service
  ExecStart=/usr/bin/cri-dockerd --network-plugin=cni   --pod-infra-container-image=registry.aliyuncs.com/google_containers/pause:3.9
  ## 启动cri
  systemctl daemon-reload && systemctl restart cri-docker.service
  查看状态
  systemctl status cri-docker.service

  ## 启动docker
  systemctl start  docker
  systemctl enable  docker
  ```

## 安装 ​`kubelet kubeadm kubectl`​

* RHEL系统

  **配置 yum 源**

  官网提供的 google 源一般用不了，这里直接换成阿里的源：

  ```shell
  cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
  [kubernetes]
  name=Kubernetes
  baseurl=http://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64
  enabled=1
  gpgcheck=1
  repo_gpgcheck=1
  gpgkey=http://mirrors.aliyun.com/kubernetes/yum/doc/yum-key.gpg
          http://mirrors.aliyun.com/kubernetes/yum/doc/rpm-package-key.gpg
  exclude=kubelet kubeadm kubectl
  EOF

  ```

  然后执行安装

  ```shell
  # --disableexcludes 禁掉除了kubernetes之外的别的仓库
  # 由于官网未开放同步方式, 替换成阿里源后可能会有索引 gpg 检查失败的情况, 这时请带上`--nogpgcheck`选项安装
  # 指定安装 1.23.5 版本
  # sudo yum install -y kubelet-1.23.5 kubeadm-1.23.5 kubectl-1.23.5 --disableexcludes=kubernetes --nogpgcheck
  # 我们直接安装最新版本
  yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes
  systemctl enable --now kubelet
  ## 查看版本
  kubeadm  version
  ##  kubeadm 和 kubectl 命令 bash 自动补全。
  kubeadm completion bash > /etc/bash_completion.d/kubeadm
  kubectl completion bash >/etc/bash_completion.d/kubectl
  source /etc/bash_completion.d/kubeadm
  source /etc/bash_completion.d/kubectl
  systemctl enable kubelet ; systemctl start kubelet
  ```

‍

* Ubuntu系统

  配置阿里源，安装包

  ```shell
  apt-get update && apt-get install -y apt-transport-https ca-certificates curl gpg
  curl https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg | apt-key add - 
  cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
  deb https://mirrors.aliyun.com/kubernetes/apt/ kubernetes-xenial main
  EOF
  apt-get update
  apt-get install -y kubelet kubeadm kubectl
  apt-mark hold kubelet kubeadm kubectl
  ## 查看版本
  kubeadm  version
  ##  kubeadm 和 kubectl 命令 bash 自动补全。
  kubeadm completion bash > /etc/bash_completion.d/kubeadm
  kubectl completion bash >/etc/bash_completion.d/kubectl
  source /etc/bash_completion.d/kubeadm
  source /etc/bash_completion.d/kubectl
  systemctl enable kubelet ; systemctl start kubelet
  ```

‍

## 下载 kubernetes 相关镜像

使用 kubeadm 工具安装，会自动从 k8s.gcr.io 下载 k8s 的相关镜像。但是此站点位于国外会出现无法访问或者速度缓慢的情况，会出现拉取不到镜像导致安装失败的情况。  
因此我们提前准备好镜像，可以是提前下载好导入、使用其它国内镜像源、自建仓库拉取镜像。

```shell
root@localhost:~# kubeadm config images list 
registry.k8s.io/kube-apiserver:v1.28.4
registry.k8s.io/kube-controller-manager:v1.28.4
registry.k8s.io/kube-scheduler:v1.28.4
registry.k8s.io/kube-proxy:v1.28.4
registry.k8s.io/pause:3.9
registry.k8s.io/etcd:3.5.9-0
registry.k8s.io/coredns/coredns:v1.10.1
```

换成阿里的库

```shell
kubeadm config images list --image-repository=registry.aliyuncs.com/google_containers
## 如果想指定版本则
## kubeadm config images list --kubernetes-version 1.22.42 --image-repository=registry.aliyuncs.com/google_containers

```

‍

## 开始初始化

* container

  ```shell
  kubeadm init --image-repository=registry.aliyuncs.com/google_containers   --kubernetes-version=v1.28.4 --service-cidr=10.1.0.0/16 --pod-network-cidr=10.244.0.0/16

  ```

* docker执行如下

  ```shell
  ## 此命令container也可以用，最最后的--cri-socket=要删除
  kubeadm init   --apiserver-advertise-address=192.168.31.52  --apiserver-bind-port=6443 --kubernetes-version=1.28.4  --pod-network-cidr=10.200.0.0/16 --service-cidr=192.168.3.0/24 --service-dns-domain=cluster.local  --image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers --ignore-preflight-errors=swap   --cri-socket=unix:///run/cri-dockerd.sock
  ```

  ‍

master执行后的结果

```shell
root@localhost:~# kubeadm init --image-repository=registry.aliyuncs.com/google_containers   --kubernetes-version=v1.28.4 --service-cidr=10.1.0.0/16 --pod-network-cidr=10.244.0.0/16
[init] Using Kubernetes version: v1.28.4
[preflight] Running pre-flight checks
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
W1119 16:56:12.408142    7613 checks.go:835] detected that the sandbox image "registry.k8s.io/pause:3.8" of the container runtime is inconsistent with that used by kubeadm. It is recommended that using "registry.aliyuncs.com/google_containers/pause:3.9" as the CRI sandbox image.
## 正在下载镜像ing...
##最后
kubeadm join 192.168.31.52:6443 --token uqixko.jbbwnnow2hy6penf \
        --discovery-token-ca-cert-hash sha256:1fb83dca38625c696b1decc6e988162f255b94f4003706b7f81315f49aa9af20 
```

## 其它节点操作

重复上面安装镜像仓库和kubelet kubeadm kubectl后

执行

```shell
kubeadm join 192.168.31.52:6443 --token uqixko.jbbwnnow2hy6penf \
        --discovery-token-ca-cert-hash sha256:1fb83dca38625c696b1decc6e988162f255b94f4003706b7f81315f49aa9af20 
```

‍

## 异常

1. api获取失败

```shell
root@kubeadm-3:~# kubectl get nodes
E1120 09:34:41.846805   45388 memcache.go:265] couldn't get current server API group list: Get "http://localhost:8080/api?timeout=32s": dial tcp 127.0.0.1:8080: connect: connection refused
E1120 09:34:41.846986   45388 memcache.go:265] couldn't get current server API group list: Get "http://localhost:8080/api?timeout=32s": dial tcp 127.0.0.1:8080: connect: connection refused
E1120 09:34:41.848196   45388 memcache.go:265] couldn't get current server API group list: Get "http://localhost:8080/api?timeout=32s": dial tcp 127.0.0.1:8080: connect: connection refused
E1120 09:34:41.849385   45388 memcache.go:265] couldn't get current server API group list: Get "http://localhost:8080/api?timeout=32s": dial tcp 127.0.0.1:8080: connect: connection refused
E1120 09:34:41.850597   45388 memcache.go:265] couldn't get current server API group list: Get "http://localhost:8080/api?timeout=32s": dial tcp 127.0.0.1:8080: connect: connection refused
The connection to the server localhost:8080 was refused - did you specify the right host or port?
```

解决: 客户端未配置

```shell
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

#然后重新kubectl get node正常
root@kubeadm-1:~# kubectl get node
NAME        STATUS     ROLES           AGE   VERSION
kubeadm-1   NotReady   control-plane   22m   v1.28.2
kubeadm-2   NotReady   <none>          10m   v1.28.2
kubeadm-3   NotReady   <none>          10m   v1.28.2
```

2. 节点状态为NotReady，

```shell
root@kubeadm-1:/etc/containerd# systemctl  status kubelet
#发现报错
Nov 20 09:53:02 kubeadm-1 kubelet[53227]: E1120 09:53:02.813389   53227 kubelet.go:2855] "Container runtime network not ready" networkReady="NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized"
```

这个问题是没有安装网络插件导致的，安装一下就好了

```shell
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
```

‍

3. 启动后，有节点是NotRead

```shell
root@kubeadm-1:~# kubectl get node 
NAME        STATUS     ROLES           AGE   VERSION
kubeadm-1   Ready      control-plane   25m   v1.28.2
kubeadm-2   Ready      control-plane   17m   v1.28.2
kubeadm-3   Ready      control-plane   16m   v1.28.2
kubeadm-4   Ready      <none>          15m   v1.28.2
kubeadm-5   NotReady   <none>          26s   v1.28.2
```

在kubeadm-5节点执行如下命令查询

```shell
root@kubeadm-5:/etc/containerd#  journalctl  -f -u  kubelet.service
Nov 22 09:35:12 kubeadm-5 kubelet[103395]: I1122 09:35:12.296389  103395 pod_startup_latency_tracker.go:102] "Observed pod startup duration" pod="kube-system/kube-proxy-7jdv2" podStartSLOduration=-5.703649857 podCreationTimestamp="2023-11-22 09:35:18 +0000 UTC" firstStartedPulling="0001-01-01 00:00:00 +0000 UTC" lastFinishedPulling="0001-01-01 00:00:00 +0000 UTC" observedRunningTime="2023-11-22 09:35:12.295937365 +0000 UTC m=+486.211870332" watchObservedRunningTime="2023-11-22 09:35:12.296350143 +0000 UTC m=+486.212283077"
Nov 22 09:35:12 kubeadm-5 kubelet[103395]: I1122 09:35:12.390389  103395 kubelet_volumes.go:161] "Cleaned up orphaned pod volumes dir" podUID="50d8e5cb-ba26-4752-9150-939d408dbceb" path="/var/lib/kubelet/pods/50d8e5cb-ba26-4752-9150-939d408dbceb/volumes"
Nov 22 09:35:16 kubeadm-5 kubelet[103395]: E1122 09:35:16.604485  103395 kubelet.go:2855] "Container runtime network not ready" networkReady="NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized"
## 报了cni  plugin未初始化
```

我们从kubeadm-1服务器，同步一份cni过来

```shell
scp -r /etc/cni/net.d/*  kubeadm-5:/etc/cni/net.d/
```

再查看状态

```shell
root@kubeadm-1:~# kubectl get node 
NAME        STATUS   ROLES           AGE   VERSION
kubeadm-1   Ready    control-plane   40m   v1.28.2
kubeadm-2   Ready    control-plane   32m   v1.28.2
kubeadm-3   Ready    control-plane   30m   v1.28.2
kubeadm-4   Ready    <none>          30m   v1.28.2
kubeadm-5   Ready    <none>          15m   v1.28.2
## 服务正常
```
