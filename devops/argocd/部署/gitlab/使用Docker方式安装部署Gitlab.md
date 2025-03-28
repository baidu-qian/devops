# 使用Docker方式安装部署Gitlab

## 背景

```shell
部署一套gitlab
```

## 参考

```shell
https://developer.aliyun.com/article/922952
```

## 部署

### 服务器

|id|cpu|内存|硬盘|操作系统|
| ----| -----| ------| ------| --------------|
|1|2|16G|20G|ubuntu 22.04|
||||||

### 部署

1. 安装docker

    ```shell
    curl -fsSL http://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository "deb [arch=amd64] http://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable"
    apt-get install docker-ce docker-ce-cli containerd.io docker-compose

    ```

2. 安装镜像

    ```shell
    docker pull gitlab/gitlab-ce

    ```

3. 编写docker-compose

    ```shell
    version: '3.8'

    services:
        gitlab:
            image: "gitlab/gitlab-ce"
            restart: always
            #privileged: true
            #user: root
            environment:
                TZ: PRC
                GITLAB_OMNIBUS_CONFIG: |
                  gitlab_rails['time_zone'] = 'Asia/Shanghai'
                  gitlab_rails['backup_keep_time'] = 259200 # 3 Day, 259200 seconds
                  #gitlab_rails['initial_root_password'] = '12345678'
                  gitlab_rails['gitlab_ssh_host']='192.168.31.221'
                  gitlab_rails['gitlab_shell_ssh_port']='30022'
                  prometheus_monitoring['enable'] = false
                  puma['worker_processes'] = 0
                  sidekiq['max_concurrency'] = 10
                GITLAB_TIMEZONE: gitlab_rails['time_zone'] = 'Beijing'
            ports:
                - 80:80/tcp
                - 443:443/tcp
                - 30022:22/tcp
            volumes:
                - /data/gitlab/data:/var/opt/gitlab
                - /data/gitlab/logs:/var/log/gitlab
                - /data/gitlab/config:/etc/gitlab
            logging:
                driver: "json-file"
                options:
                    max-size: "10m"
                    max-file: "3"
    ```

4. 启动服务

```shell
docker-compose  up -d 
```

5. 修改gitlab.rb文件

```shell
vim /data/gitlab/config/gitlab.rb
```

修改如下位置：

```
# 如果使用公有云且配置了域名了，可以直接设置为域名，如下
external_url 'http://gitlab.redrose2100.com'
# 如果没有域名，则直接使用宿主机的ip，如下
external_url 'http://172.22.27.162'  
```

```
# 同样如果有域名，这里也可以直接使用域名
gitlab_rails['gitlab_ssh_host'] =  'gitlab.redrosee2100.com'
# 同样如果没有域名，则直接使用宿主机的ip地址
gitlab_rails['gitlab_ssh_host'] = '172.22.27.162'
```

```
# 端口为启动docker时映射的ssh端口
gitlab_rails['gitlab_shell_ssh_port'] =10010 
```

```
# 设置时区为东八区，即北京时间
gitlab_rails['time_zone'] = 'Asia/Shanghai'  
```

关于邮箱发邮件的配置如下

```
gitlab_rails['smtp_enable'] = true
gitlab_rails['smtp_address'] = "smtp.163.com"   # 邮箱服务器
gitlab_rails['smtp_port'] = 465    # 邮箱服务对应的端口号
gitlab_rails['smtp_user_name'] = "hitredrose@163.com"   # 发件箱的邮箱地址
gitlab_rails['smtp_password'] = "xxxxxxxxxxx"      # 发件箱对应的授权码，注意不是登录密码，是授权码
gitlab_rails['smtp_domain'] = "163.com"
gitlab_rails['smtp_authentication'] = "login"
gitlab_rails['smtp_enable_starttls_auto'] = true
gitlab_rails['smtp_tls'] = true
gitlab_rails['gitlab_email_enabled'] = true
gitlab_rails['gitlab_email_from'] = 'hitredrose@163.com'     # 发件箱地址
gitlab_rails['gitlab_email_display_name'] = 'gitlab.redrose2100.com'    # 显示名称
gitlab_rails['gitlab_email_reply_to'] = 'noreply@example.com'     # 提示不要回复
```

6. 重启docker

    ```shell
    docker-compose down 
    docker-compose up -d 
    ```

‍
