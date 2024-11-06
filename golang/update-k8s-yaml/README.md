# 作用
用于升级更新k8s的部署文件中，镜像名，变量值等信息，方便在离线交付，不能用helm的情况下，批量进行替换关系的配置信息
脚本分别用golang和python3来实现

# 用法
``` shell 
vim config/config.yaml
# 修改需要调整的配置信息
go run update-k8s-yaml.go
# 替换成功后，查看文件
```
