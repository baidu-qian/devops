'''
Author: magician
Date: 2024-07-09 00:36:55
LastEditors: magician
LastEditTime: 2024-07-09 00:43:00
FilePath: /python/运维/hbase/hbase.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
import requests

def api_get_call(url):
    try:
        response = requests.get(url, timeout=10)
    except:
        # print("ERROR: failed to make an API call:", url)
        return False
    if response.status_code != 200 and response.status_code != 202:
        # print("ERROR:", response.json()["error"])
        return False
        # raise Exception("failed to make an API call, %s, %s" % (response.status, response.reason))
    return True


if __name__ == "__main__":
    url="http://172.16.44.18:60010"
    if api_get_call(url):
        print("连接成功")
    else:
        print("连接失败")
