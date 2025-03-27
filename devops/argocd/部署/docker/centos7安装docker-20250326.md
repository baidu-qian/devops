# centos7安装docker-20250326

1. 设置yum源

    ```shell
    yum-config-manager --add-repo http://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo（阿里仓库）
    ```

2. 安装docker-ce

    ```shell
    yum install -y  docker-ce
    ```

3. 调整docker的配置文件

    ```shell
    cat > /etc/docker/daemon.json < EOF
    {
      "data-root": "/data/docker",
      "builder": {
        "gc": {
          "defaultKeepStorage": "20GB",
          "enabled": true
        }
      },
      "experimental": false,
      "insecure-registries": [
        "http://172.16.44.141"
      ],
      "log-driver": "json-file",
      "log-opts": {
        "max-size": "100m"
      },
      "registry-mirrors": [
        "https://dockerpull.cn",
        "https://dockerpull.pw"
      ]
    }
    EOF
    ```

4. 重启docker

    ```shell
    systemctl restart docker
    systemctl enable docker
    ```

5. 安装docker-compose

    ```shell
    wget https://github.ednovas.xyz/https://github.com/docker/compose/releases/download/v2.34.0/docker-compose-linux-x86_64
    mv docker-compose-linux-x86_64  docker-compose
    chmod +x docker-compose
    mv docker-compose /usr/local/bin/
    ```

6. 检查

    ```shell
    docker  ps  
    docker-compose 
    # 命令都有返回值 
    ```
