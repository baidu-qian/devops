# 作用
用于根据kafka的堆积情况，自动进行扩容服务
需要服务通过docker-compose.yaml进行部署，且docker-compose定义了如下副本内容 
``` shell
    deploy:
      endpoint_mode: vip
      mode: replicated
      replicas: 1
```


# 用法
``` shell 
vim config/config.yaml
# 修改配置文件信息
python3  auto-replicas.py
```
