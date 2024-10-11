# 作用
用于快速搭建kerberos环境，脚本用于编译kerberos镜像+docker-compose快速运行

# 使用说明 
## 编译
docker build -t kerberos:1.0.0 .

## 运行
1. 将docker-compose目录中的kerberos目录推送至目标服务器
2. 安装docker环境,安装docker-compose,没有来`https://github.com/docker/compose/releases`下载安装
3. 加载镜像
4. kerberos/bin/docker-compose.yaml 中的路径调整为自己的路径
5. cd kerberos/bin;  docker-compose up -d 
6. 验证
