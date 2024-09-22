# 作用
启动一个http端口8080，通过浏览器访问，返回当前主机的信息

# 用法
``` shell 
vim http.py
# 修改端口，默认是8080
python3 http.go
```

## 返回值
我自己电脑测试返回值为
``` json
{
  "hostname": "MacBook-Pro-3.local",
  "ip_address": "::1",
  "current_time": "2024-09-22T21:35:58+08:00"
}
```
