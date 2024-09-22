'''
Author: hongchun.you
Date: 2023-10-20 15:58:13
LastEditors: magician
LastEditTime: 2024-08-24 16:01:08
FilePath: /python/es/query/weixin.py
Description: 
Copyright (c) 2023 by hongchun.you, All Rights Reserved. 
'''
from elasticsearch import Elasticsearch
from elasticsearch.helpers import scan,bulk
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import ThreadPoolExecutor
from kafka import KafkaConsumer, TopicPartition
from kafka import KafkaAdminClient
from kafka.structs import TopicPartition

from kazoo.client import KazooClient
import kazoo.exceptions

import smtplib
from email.mime.text import MIMEText

from flask import Flask, jsonify
import socket

import time
import requests
import json
import os
import psycopg2
import redis

import yaml

def indices_count(es,indices,username,password):
    es = Elasticsearch(
        [es],  # Elasticsearch 服务器的地址和端口
        http_auth=(username, password),  # 如果需要身份验证，请提供用户名和密码
        timeout=30,  # 连接超时时间
        max_retries=5,  # 最大重试次数
        retry_on_timeout=True  # 在超时时重试
    )
    # response = es.transport.perform_request("GET", "/_cat/count")
    total_count = es.cat.count(indices)
    lines = total_count.split('\n')
    count = int(lines[0].split()[2])  # 解析响应，获取文档总数
    return count

def validate_index(es, username, password, sleep_time, index_name):
    initial_total_count = indices_count(es, index_name, username, password)
    time.sleep(sleep_time)
    final_total_count = indices_count(es, index_name, username, password)
    if final_total_count > initial_total_count:
        # return f"{index_name}: 文档总数增加"
        return  None
    else:
        # return f"{index_name}: 文档总数未变化或减少"
        return  index_name

# 发送文本消息
def send_text(webhook, content, mentioned_list=None, mentioned_mobile_list=None):
    header = {
                "Content-Type": "application/json",
                "Charset": "UTF-8"
                }
    data ={

        "msgtype": "text",
        "text": {
            "content": content
            ,"mentioned_list":mentioned_list
            ,"mentioned_mobile_list":mentioned_mobile_list
        }
    }
    data = json.dumps(data)
    info = requests.post(url=webhook, data=data, headers=header)
    

# 发送markdown消息
def send_md(webhook, content):
    header = {
                "Content-Type": "application/json",
                "Charset": "UTF-8"
                }
    data ={
        "msgtype": "markdown",
        "markdown": {
            "content": content
        }
    }
    data = json.dumps(data)
    info = requests.post(url=webhook, data=data, headers=header)

def api_get_call(url,headers,auth):
    try:
        response = requests.get(url, headers=headers, auth=auth)
    except:
        # print("ERROR: failed to make an API call:", url)
        return "error"
    if response.status_code != 200 and response.status_code != 202:
        # print("ERROR:", response.json()["error"])
        return "error"
        # raise Exception("failed to make an API call, %s, %s" % (response.status, response.reason))
    return response.json()

# nginx及hbase等服务检测
def api_get_call_nginx(url):
    try:
        response = requests.get(url, verify=False,timeout=10)
    except:
        return False
    if response.status_code != 200 and response.status_code != 202:
        return False
    return True

def check_service_status(config,header,auth,service_name):
    service_list = config.get(service_name,[])
    status_list = []
    for service in service_list:
        if service["enable"]:
            ip = service.get("ip")
            port = service.get("port")
            url = f"http://{ip}:{port}/actuator/health"
            # app_status = api_get_call(url, header, auth)
            # if app_status == "error":
                # status_list.append(service_name)
                # continue
            isUp = False
            for i in range(3):
                app_status = api_get_call(url, header, auth)
                if app_status != "up":
                    time.sleep(5)
                    if i == 2:
                        isUp = False
                    # status_list.append(service_name)
                    # continue
                else:
                    isUp = True
                    break
            if not isUp:
                status_list.append(service_name)
    return status_list


def check_all_services(config):
    header = {
                "Content-Type": "application/json",
                "Charset": "UTF-8"
                }
    auth = (config["prometheus"].get("username"), config["prometheus"].get("password"))
    services_to_check = ["receiver", "cleaner", "security-event", "threat", "threat-index", "transfer", "web-service"]
    app_status_list = []
    
    # 使用 ThreadPoolExecutor 来并发验证
    # for service_name in services_to_check:
    with ThreadPoolExecutor(max_workers=len(services_to_check)) as executor:
    # with ThreadPoolExecutor(max_workers=999) as executor:
        # for service_name in services_to_check:
        futures = [executor.submit(check_service_status, config, header, auth, service_name) for service_name in  services_to_check]
        # 收集结果
        results = [future.result() for future in futures]

    # 打印验证结果
    for result in results:
        if result is not None:
            app_status_list.extend(result)
                # print(result)
    # for service_name in services_to_check:
    #     status_list = check_service_status(config, header, auth, service_name)
    #     app_status_list.extend(status_list)
    
    return app_status_list


def prometheus(config):
    app_status_list = []
    header = {
                "Content-Type": "application/json",
                "Charset": "UTF-8"
                }
    auth = (config["prometheus"].get("username"), config["prometheus"].get("password"))
    receiver = config.get("receiver")
    cleaner = config.get("cleaner")
    security_event = config.get("security-event")
    threat = config.get("threat")
    threat_index = config.get("threat-index")
    transfer = config.get("transfer")
    web = config.get("web-service")
    for re in receiver:
        ip = re["ip"]
        port = re["port"]
        print(f"Receiver - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("receiver")
    for cl in cleaner:
        ip = cl["ip"]
        port = cl["port"]
        print(f"cleaner - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("cleaner")
    for se in security_event:
        ip = se["ip"]
        port = se["port"]
        print(f"security_event - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("security_event")
    for th in threat:
        ip = th["ip"]
        port = th["port"]
        print(f"threat - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("threat")
    for thi in threat_index:
        ip = thi["ip"]
        port = thi["port"]
        print(f"threat-index - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("threat_index")
    for tr in transfer:
        ip = tr["ip"]
        port = tr["port"]
        print(f"transfer - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("transfer")
    for we in web:
        ip = we["ip"]
        port = we["port"]
        print(f"web - IP: {ip}, Port: {port}")
        url = f"http://{ip}:{port}/actuator/health"
        app_status = api_get_call(url,header,auth)
        print(f"App Status: {app_status['status']}")
        if "status" in app_status and app_status["status"] != "UP":
            app_status_list.append("web")
    return app_status_list

def get_consumer_lag(kafka_url,topic,group_id,lag=100000):
    # 配置 Kafka 消费者
    consumer = KafkaConsumer(
        # 'app_information',  # 替换为您要监控的 Kafka 主题
        group_id=group_id,  # 替换为您的 Kafka 消费者组
        bootstrap_servers=kafka_url,  # 替换为您的 Kafka 服务器地址
    )
    # 获取分区列表
    partitions = consumer.partitions_for_topic(topic)
    if partitions is None:
        raise ValueError("Topic not found")
    
    # 初始化lag为0
    lags = 0
    # 遍历每个分区
    for partition in partitions:
        tp = TopicPartition(topic, partition)

        # 分配分区并获取分区结束位置
        consumer.assign([tp])
        consumer.seek_to_end(tp)
        end_offset = consumer.position(tp)

        # 获取分区的 CURRENT-OFFSET（消费者当前位置）
        consumer.seek_to_beginning(tp)
        current_offset = consumer.position(tp)

        # 计算 LAG（滞后）
        kafka_lag = end_offset - current_offset
        lags += kafka_lag
    if lags > lag:
        # print(f"Topic: {topic}, Partition: {partition}, CURRENT-OFFSET: {current_offset}, LOG-END-OFFSET: {end_offset}, LAG: {lag}")
        return(f"GroupId: {group_id},Topic: {topic}, LAGS总数: {lags}")
        # return(group_id,topic,lags)
    # 关闭 Kafka 消费者
    consumer.close()

def get_kafka_msg(kafka_url,lag):
    # 配置 Kafka 集群的连接信息
    try:
        admin_client = KafkaAdminClient(
            bootstrap_servers=kafka_url,  # 替换为您的 Kafka 服务器地址
        )
    except Exception as e:
        return [f"kafka连接失败:{e}"]
    # 获取 Kafka 集群中的所有主题
    topics = admin_client.list_topics()
    # print("所有主题:")
    # for topic in topics:
    #     print(topic)


    # 获取消费者组信息
    consumer_groups = admin_client.list_consumer_groups()
    # print("所有消费者组:")
    # for consumer_group in consumer_groups:
    #     print(consumer_group[0])

    # # 获取每个消费者组订阅的主题
    # print("消费者组和它们订阅的主题:")
    # for consumer_group in consumer_groups:
    #     group_description = admin_client.list_consumer_groups(consumer_group[0])
    #     # 打印消费者组和其订阅的主题
    #     print(f"Consumer Group: {consumer_group[0]}, Subscribed Topics: {group_description}")
    #     # for i in group_description.topics:
    #         # print(f"Consumer Group: {consumer_group[0]}, Subscribed Topics: {i}")

    unique_topics = set()
    result_seds = []
    # 遍历每个消费者组
    for group in consumer_groups:
        # 获取消费者组订阅的主题
        topics = admin_client.list_consumer_group_offsets(group[0])
        # 处理返回的 TopicPartition 列表
        
        for tp in topics:
            # topic_name = tp.topic
            # print(f"Consumer Group: {group[0]}, Subscribed Topic: {topic_name}")
            topic_name = tp.topic
            unique_topics.add(topic_name)
        # 打印唯一的主题名称
        # for topic_name in unique_topics:
        #     # print(f"Consumer Group: {group[0]}, Subscribed Topic: {topic_name}")
        #     get_consumer_lag(kafka_url,topic_name,group[0],limit)
        # 使用 ThreadPoolExecutor 来并发验证
        with ThreadPoolExecutor(max_workers=len(unique_topics)) as executor:
        # with ThreadPoolExecutor(max_workers=999) as executor:
            futures = [executor.submit(get_consumer_lag, kafka_url,topic_name,group[0],lag) for topic_name in unique_topics]
            # 收集结果
            results = [future.result() for future in futures]

        # 打印验证结果
        for result in results:
            if result is not None:
                result_seds.append(result)
                # print(result)
    return result_seds

def check_postgres_status(ip,port,database,username,password):
    try:
        conn = psycopg2.connect(database=database, user=username, password=password, host=ip, port=port)
        # 获得游标对象
        cursor = conn.cursor()
        # 执行一个查询 
        cursor.execute("SELECT 1")
        # 获取查询结果
        result = cursor.fetchone()
        # 如果查询成功，则result 返回(1,)
        if result == (1,):
            return True
        else:
            return False
    except psycopg2.Error as e:
        # print(e)
        return False

def check_redis_status(ip,port,password):
    try:
        r = redis.Redis(host=ip, port=port, password=password)
        r.ping()
        return True
    except Exception as e:
        print(f"Redis连接失败: {str(e)}")
        return False

def check_zookeepr_status(hosts="172.16.51.110:2181"):
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

def send_email(subject, message, from_addr, password, to_addr, smtp_server='smtp.qq.com', port=465):
    # 创建邮件内容
    msg = MIMEText(message)
    msg['Subject'] = subject
    msg['From'] = from_addr
    msg['To'] = to_addr

    try: 
        # 如果使用 Gmail 作为 SMTP 服务器，需要开始 TLS 安全连接
        server = smtplib.SMTP_SSL(smtp_server, port,timeout=10)
        # 登录到 SMTP 服务器
        server.login(from_addr, password)  # 注意：这里的密码需要替换为实际的密码
        # 发送邮件
        server.send_message(msg)
        server.quit()
    except Exception as e:
        print(f"邮件发送失败: {str(e)}")
        try:
            # 如果 SSL 连接失败，尝试使用 TLS 连接
            server = smtplib.SMTP(smtp_server, port,timeout=10)
            server.starttls()  # 启用 TLS
            server.login(from_addr, password)  # 替换为实际密码
            server.send_message(MIMEText(message))
            print("Email sent via TLS")
        except Exception as e:
            print(f"TLS connection failed: {e}")

# 是否为生产环境
production_tag = False

def main(config):
    # es = "http://172.16.44.69:9200"
    # es = "http://123.56.80.234:18013/"
    # # 指定 Elasticsearch 服务器和验证参数
    # es = Elasticsearch(
    #     [es],  # Elasticsearch 服务器的地址和端口
    #     http_auth=(es_username, es_password),  # 如果需要身份验证，请提供用户名和密码
    #     timeout=30,  # 连接超时时间
    #     max_retries=5,  # 最大重试次数
    #     retry_on_timeout=True  # 在超时时重试
    # )

    # 指定要验证的参数列表
    app_list = ["app", "crash", "dev_status", "devinfo", "env", "msg", "security_event", "start", "threat", "threatindex", "user_data"]

    # 构建索引名称列表
    index_names = ["bb_i_" + i + "*" for i in app_list]

    if production_tag:
        sleep_time = 300
    else:
        sleep_time = 5
    
    while True:
        result_seds = []
        if config["database"]["es"]["enable"]:
            # 使用 ThreadPoolExecutor 来并发验证
            with ThreadPoolExecutor(max_workers=len(index_names)) as executor:
                futures = [executor.submit(validate_index, config["database"]["es"]["url"], config["database"]["es"]["username"], config["database"]["es"]["password"], sleep_time, index_name) for index_name in index_names]

                # 收集结果
                results = [future.result() for future in futures]

            # 打印验证结果
            # result_seds = []
            for result in results:
                # print(result)
                if result:
                    # print(result)
                    result_seds.append(result)
                    # send_md(webhook, content="# 威胁感知告警: \n 威胁感知es表:"+result+"最近5分钟未增长!")
                else:
                    print("None:"+result)
        send_weixin = ""
        # 如果result_seds被定义
        
        if len(result_seds) != 0:
            send_msg = "# 威胁感知ES告警: \n"
            for index, value in enumerate(result_seds):
                # print(str(index+1) + ". 威胁感知es表:"+value+"最近5分钟未增长! \n")
                send_msg = send_msg + str(index+1)+". 威胁感知es表:"+value+"最近5分钟未增长! \n"
            # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
            send_weixin = send_weixin + send_msg

        # 在所有索引app_list后，等待 10 秒
        # time.sleep(10)
        # 访问配置项
        # app_list = prometheus(config["database"])
        app_list = check_all_services(config["database"])
        if len(app_list) != 0:
            send_msg = "# 威胁感知APP告警: \n"
            for index, value in enumerate(app_list):
                # print(str(index+1) + ". 威胁感知app:"+value+"当前状态为UP! \n")
                send_msg = send_msg + str(index+1)+". 威胁感知app:"+value+"当前状态不为UP! \n"
            # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
            send_weixin = send_weixin + send_msg
        # kafka数据推送
        # kafka_msg = get_kafka_msg(config["database"]["kafka"]["lag"], config["database"]["kafka"]["lag"])
        if config["database"]["kafka"]["enable"]:
            kafka_msg = get_kafka_msg(config["database"]["kafka"]["brokers"], config["database"]["kafka"]["lag"])
            if len(kafka_msg) != 0:
                send_msg = "# 威胁感知kafka告警: \n"
                for index, value in enumerate(kafka_msg):
                    print(str(index+1) + ". 威胁感知kafka:"+value+"\n")
                    send_msg = send_msg + str(index+1)+". 威胁感知kafka:"+value+"\n"
                # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                # send_msg = send_msg + kafka_msg + "\n"
                send_weixin = send_weixin + send_msg 
        # postgres 数据库状态
        service_list = config["database"]["postgres"]
        for service in service_list:
            if service["enable"]:
                postgres_status = check_postgres_status(service["ip"],service["port"],service["database"],service["username"],service["password"])
                if postgres_status == False:
                    pg_msg = "# 威胁感知Postgres告警: \n"
                    pg_msg = pg_msg + f"Postgres: {service['ip']}:{service['port']} 数据库连接失败! \n"
                    # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                    send_weixin = send_weixin + pg_msg
        # redis 数据库状态
        service_list = config["database"]["redis"]
        for service in service_list:
            if service["enable"]:
                redis_status = check_redis_status(service["ip"],service["port"],service["password"])
                if redis_status == False:
                    redis_msg = "# 威胁感知Redis告警: \n"
                    redis_msg = redis_msg + f"Redis: {service['ip']}:{service['port']} 数据库连接失败! \n"
                    # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                    send_weixin = send_weixin + redis_msg
        # zookeeper 数据库状态
        service_list = config["database"]["zookeeper"]
        for service in service_list:
            if service["enable"]:
                zookeeper_status = check_zookeepr_status(service["ip"]+":"+str(service["port"]))
                if zookeeper_status == False:
                    zookeeper_msg = "# 威胁感知Zookeeper告警: \n"
                    zookeeper_msg = zookeeper_msg + f"Zookeeper: {service['ip']}:{service['port']} 数据库连接失败! \n"
                    # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                    send_weixin = send_weixin + zookeeper_msg
        # hbase 服务状态
        service_list = config["database"]["hbase"]
        for service in service_list:
            if service["enable"]:
                hbase_status = api_get_call_nginx("http://%s:%d" % (service["ip"],service["port"]))
                if hbase_status == False:
                    hbase_msg = "# 威胁感知Hbase告警: \n"
                    hbase_msg = hbase_msg + f"Hbase: {service['ip']}:{service['port']} 数据库连接失败! \n"
                    # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                    send_weixin = send_weixin + hbase_msg
        # nginx服务状态
        service_list = []
        service_list.append(config["database"]["nginx"])
        service_list.append(config["database"]["web-service-nginx"])
        service_list.append(config["database"]["kibana"])
        for app_list in service_list:
            for service in app_list:
                if service["enable"]:
                    nginx_status = api_get_call_nginx("http://%s:%d" % (service["ip"],service["port"]))
                    if nginx_status == False:
                        nginx_status = api_get_call_nginx("https://%s:%d" % (service["ip"],service["port"]))
                    if nginx_status == False:
                        nginx_msg = "# 威胁感知HTTP服务告警: \n"
                        nginx_msg = nginx_msg + f"Nginx: {service['ip']}:{service['port']} 数据库连接失败! \n"
                        # send_md(config["database"]["weixin"]["webhook"], content=send_msg)
                        send_weixin = send_weixin + nginx_msg
        
        if len(result_seds) != 0 or len(app_list) != 0 or len(kafka_msg) != 0 or len(pg_msg) or len(zookeeper_msg)!=0 or len(hbase_msg)!=0 or len(nginx_msg)!=0:
            if production_tag == True:
                if config["database"]["weixin"]["enable"]:
                    send_md(config["database"]["weixin"]["webhook"], content=send_weixin)
            if config["database"]["email"]["enable"]:
                subject="【威胁感知】官服邮件告警"
                send_email(subject,send_weixin,config["database"]["email"]["from_addr"],config["database"]["email"]["password"],config["database"]["email"]["to_addr"],config["database"]["email"]["smtp_server"],config["database"]["email"]["port"])
            else:
                print(send_weixin)




if __name__ == "__main__":
    if production_tag == True:
        config_file_path = "es/query/config-production.yaml"  # 替换成你的配置文件路径
    else:
        config_file_path = "es/query/config-test.yaml"
    abs_path = os.path.abspath(config_file_path)
    print(abs_path)
    # config_data = read_config_file(config_file_path)
    with open(config_file_path, "r") as yamlfile:
        config = yaml.safe_load(yamlfile)
    main(config)
