# 作用
用于检查检查业务是否异常，如果异常，进行推送告警

- 判断数据源
1. kafka是否堆积
2. es是否未增长
3. redis,pg是否可写
4. 业务服务的prometheus状态是否正常

- 推送目标
1. 企业微信
2. 邮件

- 支持开头

# 用法
``` shell 
vim  config.yaml
# 修改监控的目标配置
python3  weixin.py
```
