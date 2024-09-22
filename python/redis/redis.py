'''
Author: magician
Date: 2024-06-24 23:58:33
LastEditors: magician
LastEditTime: 2024-06-25 00:01:59
FilePath: /python/运维/redis/redis.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
import redis

def check_redis_status(host, port, password):
    try:
        r = redis.Redis(host=host, port=port, password=password)
        r.ping()
        return True
    except Exception as e:
        print(f"Redis连接失败: {str(e)}")
        return False

if __name__ == "__main__":
    host = "172.16.51.110"
    port = 6379
    password = ""
    redis_status = check_redis_status(host, port, password)
    if redis_status:
        print("Redis连接成功")
    else:
        print("Redis连接失败")
