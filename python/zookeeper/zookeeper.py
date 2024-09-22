'''
Author: magician
Date: 2024-06-29 22:33:15
LastEditors: magician
LastEditTime: 2024-07-01 00:17:07
FilePath: /python/运维/zookeeper/zookeeper.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
from kazoo.client import KazooClient
import kazoo.exceptions

def check_zookeepr_service(hosts="172.16.51.110:2181"):
    """
    检测Zookeeper服务是否正常
    :param hosts: Zookeeper服务器地址，格式为 "host1:port1,host2:port2"
    :return: 若服务正常，则返回True，否则返回False
    """
    zk = KazooClient(hosts=hosts)
    try:
        zk.start(timeout=5)
        ## 连接成功, 检查Zookeeper状态
        if zk.state == "CONNECTED":
            print("Zookeeper服务正常.")
            return True
        else:
            print("Zookeeper服务异常.",zk.state)
            return False
    except Exception as e :
        # 处理连接或其他Zookeeper错误
        print("连接Zookeeper服务失败:", e)
        return False
    finally:
        zk.stop()
        zk.close()  

if check_zookeepr_service("172.16.51.110:2181"):
    print("Zookeeper服务正常.")
else:
    print("Zookeeper服务异常.")
