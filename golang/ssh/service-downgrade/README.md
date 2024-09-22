# 作用
用于内存，omm文件，当系统即将发生血崩时，进行服务降级
需要服务通过docker-compose.yaml进行部署


# 用法
``` shell 
vim config/config.yaml
# 修改配置文件信息
go run service-downgrade.go
```
