import os,yaml
import paramiko
from io import StringIO
import time
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import ThreadPoolExecutor
from kafka import KafkaConsumer, TopicPartition
from kafka import KafkaConsumer, TopicPartition
from kafka import KafkaAdminClient
from kafka.structs import TopicPartition


def get_consumer_lag(kafka_url,topic,group_id,lag=100000,consecutive_threshold=3):
    """
    获取 Kafka 消费者组的滞后信息。

    Args:
        kafka_url (str): Kafka 服务器地址。
        topic (str): 要监控的 Kafka 主题。
        group_id (str): Kafka 消费者组 ID。
        lag (int): 滞后阈值。
        consecutive_threshold (int): 连续滞后次数阈值，需要连续大于 lag 的次数才能返回。

    Returns:
        str: 消费者组 ID，如果连续滞后次数大于等于阈值，否则返回 None。
    """
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
    
    consecutive_lags = 0
    for _ in range(consecutive_threshold):
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
        if lags >= lag:
            consecutive_lags += 1
            time.sleep(2)
        else:
            consecutive_lags = 0
    if consecutive_lags >= consecutive_threshold:
        # print(f"Topic: {topic}, Partition: {partition}, CURRENT-OFFSET: {current_offset}, LOG-END-OFFSET: {end_offset}, LAG: {lag}")
        # return(f"GroupId: {group_id},Topic: {topic}, LAGS总数: {lags}")
        return(group_id)
    
        # return(group_id,topic,lags)
    # 关闭 Kafka 消费者
    consumer.close()

def get_kafka_msg(kafka_url,lag):
    """
    获取 Kafka 消费者组的滞后信息
    
    Args:
        kafka_url (str): Kafka 服务器地址
        lag (int): 滞后阈值
    
    Returns:
        list: 消费者组 ID，如果连续滞后次数大于等于阈值，否则返回 None
    """
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
    wxgz_groups = ["groupid_threat","groupid_dataservice","event_data_preparation","cleaners_groupid","groupid_running_distribution"]
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
        if group[0] in wxgz_groups:
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

def get_remote_system_info(ip, username, password, port):
    """
    获取远程系统信息

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口

    Returns:
        tuple: CPU 信息，内存信息，磁盘信息
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        
        # 执行命令
        stdin, stdout, stderr = ssh_client.exec_command("top -b -n 1 | grep 'Cpu(s)'")
        cpuinfo = stdout.read().decode("utf-8").split(",")[3].split()[0]
        # print("cpu信息:" + cpuinfo)

        stdin, stdout, stderr = ssh_client.exec_command("free -g|grep Mem")
        meminfo = stdout.read().decode("utf-8").split()[-1]
        # print("meminfo: " + meminfo)

        stdin, stdout, stderr = ssh_client.exec_command("df -h -BG /home/app | sort -k 4 -h|tail -n 1")
        disk_output = stdout.read().decode("utf-8").split()[3].rstrip('G')
        # print("disk_output: " + disk_output)
        return cpuinfo,meminfo,disk_output

    finally:
        # 关闭连接
        ssh_client.close()

def get_app_replicas(ip, username, password, port,app_name):
    """
    获取指定应用的副本数
    :param ip: 远程服务器IP
    :param username: 用户名
    :param password: 密码
    :param port: 端口
    :param app_name: 应用名称
    :return: 应用副本数
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)

        # 使用sftp传输文件
        sftp= ssh_client.open_sftp()
        with sftp.file("/home/app/app/"+app_name+"/bin/docker-compose.yml","r") as remote_file:
            compose_data = yaml.safe_load(remote_file)
        # 更新指定服务的环境变量
        for service in compose_data["services"].values():
            try:
                # print(service["deploy"]["replicas"])
                return(service["deploy"]["replicas"])
                break
            except:
                pass
    finally:
        sftp.close()
        ssh_client.close()

def set_app_replicas(ip, username, password, port, app_name,replicas):
    """
    设置指定应用的副本数

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        app_name (str): 应用名称
        replicas (int): 副本数

    Returns:
        None: 无返回值
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)

        # 使用sftp传输文件
        sftp= ssh_client.open_sftp()
        with sftp.file("/home/app/app/"+app_name+"/bin/docker-compose.yml","r") as remote_file:
            compose_data = yaml.safe_load(remote_file)
        # 更新指定服务的环境变量
        for service in compose_data["services"].values():
            try:
                service["deploy"]["replicas"] = replicas
                break
            except:
                pass
        # 将更新后的内容写回到远程文件
        updated_data = StringIO()
        yaml.dump(compose_data, updated_data, default_flow_style=False)
        updated_data.seek(0)
        with sftp.file("/home/app/app/"+app_name+"/bin/docker-compose.yml", 'w') as remote_file:
            remote_file.write(updated_data.read())
    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def deploy_app_replicas(app):
    """
    部署应用副本数
    Args:
        app (str): 应用名称

    Returns:
        None: 无返回值
    """
    replicas_list = config["database"]["replicas"]
    username = config["database"]["ssh"]["username"]
    password = config["database"]["ssh"]["password"]
    port = config["database"]["ssh"]["port"]
    for service in config["database"]["replicas"][app]:
        if service["enable"]:
            ip = service["ip"]
            max = service["max"]
            min = service["min"]
            cpuinfo,meminfo,disk_output = get_remote_system_info(ip, username,password,port)
            if float(cpuinfo) >= 20 and int(meminfo) >= 2 and int(disk_output) >= 20:
                print(f"{ip} cpuinfo: {cpuinfo} meminfo: {meminfo} disk_output: {disk_output}")
                replicas = get_app_replicas(ip, username,password,port,app)
                print(f"扩容:{app}当前副本数:{str(replicas)}")
                if int(replicas) < int(max):
                    set_app_replicas(ip, username,password,port,app,int(replicas)+1)
                    app_restart(ip, username,password,port,app)
                replicas = get_app_replicas(ip, username,password,port,app)
                print(f"扩容:{app}当前副本数:{str(replicas)}")

def fallback_app_replicas(app):
    """
    回滚应用副本数

    Args:
        app (str): 应用名称

    Returns:
        None: 无返回值
    """
    replicas_list = config["database"]["replicas"]
    username = config["database"]["ssh"]["username"]
    password = config["database"]["ssh"]["password"]
    port = config["database"]["ssh"]["port"]
    for service in config["database"]["replicas"][app]:
        if service["enable"]:
            ip = service["ip"]
            max = service["max"]
            min = service["min"]
            replicas = get_app_replicas(ip, username,password,port,app)
            print(f"缩容:{app}当前副本数:{str(replicas)}")
            if int(replicas) > int(min):
                set_app_replicas(ip, username,password,port,app,int(replicas)-1)
                app_restart(ip, username,password,port,app)
            replicas = get_app_replicas(ip, username,password,port,app)
            print(f"缩容:{app}当前副本数:{str(replicas)}")

def app_restart(ip, username, password, port, app_name):
    """
    应用重启

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        app_name (str): 应用名称

    Returns:
        None: 无返回值
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        
        # 执行命令
        stdin, stdout, stderr = ssh_client.exec_command("/home/app/bin/docker-compose -f /home/app/app/"+app_name+"/bin/docker-compose.yml down ")
        stdout.channel.recv_exit_status()  # 等待命令执行完成
        # time.sleep(5)
        # print("停止信息:" + stdout)
        stdin, stdout, stderr = ssh_client.exec_command("/home/app/bin/docker-compose -f /home/app/app/"+app_name+"/bin/docker-compose.yml up -d")
        stdout.channel.recv_exit_status()  # 等待命令执行完成
        # print("启动信息:" + stdout)

    finally:
        ssh_client.close()

def main(config):
    replicas_tag = 0
    # replicas_list = config["database"]["replicas"]
    # username = config["database"]["ssh"]["username"]
    # password = config["database"]["ssh"]["password"]
    # port = config["database"]["ssh"]["port"]
    # for service_list in replicas_list:
    #     for service in replicas_list.get(service_list):
    #         if service["enable"]:
    #             ip = service["ip"]
    #             cpuinfo,meminfo,disk_output = get_remote_system_info(ip, username,password,port)
    #             print(f"{ip} service_name: {service_list}  cpuinfo: {cpuinfo} meminfo: {meminfo} disk_output: {disk_output}")

    # for service_list in replicas_list:
    #     for service in replicas_list.get(service_list):
    #         if service["enable"]:
    #             ip = service["ip"]
    #             min = service["min"]
    #             max = service["max"]
    #             print(f"{ip} service_name: {service_list} min: {min} max: {max}")
    while True:
        kafka_msg = get_kafka_msg(config["database"]["kafka"]["brokers"], config["database"]["kafka"]["lag"])
        if len(kafka_msg) != 0:
            replicas_tag = 0
            unique_topics = set()
            for  index, value in enumerate(kafka_msg):
                unique_topics.add(value)
            for i in unique_topics:
                match i:
                    case "cleaners_groupid":
                        deploy_app_replicas("cleaner")
                    case "groupid_dataservice":
                        deploy_app_replicas("transfer")
                    case "event_data_preparation":
                        deploy_app_replicas("security-event")
                    case "groupid_threat":
                        deploy_app_replicas("threat")
                    case "groupid_running_distribution":
                        deploy_app_replicas("analyzer-dev")
                    case _:
                        print("未知主题,无需扩容")
        else:
            replicas_tag += 1
            if replicas_tag >= 3:
                wxgz_groups = ["cleaner", "transfer", "security-event", "threat", "analyzer-dev"]
                unique_topics = set()
                for  index, value in enumerate(kafka_msg):
                    unique_topics.add(value)
                fallback_app_tag = []
                for i in unique_topics:
                    match i:
                        case "cleaners_groupid":
                            fallback_app_tag.append("cleaner")
                        case "groupid_dataservice":
                            fallback_app_tag.append("transfer")
                        case "event_data_preparation":
                            fallback_app_tag.append("security-event")
                        case "groupid_threat":
                            fallback_app_tag.append("threat")
                        case "groupid_running_distribution":
                            fallback_app_tag.append("analyzer-dev")
                        case _:
                            print("未知主题,无需扩容")
                for app in wxgz_groups:
                    # 如果 app 不在 fallback_app_tag 中，则执行缩容操作
                    if app not in fallback_app_tag:
                        fallback_app_replicas(app)
                        time.sleep(10)  # 休眠 10 秒
                replicas_tag = 0
        time.sleep(10)

if __name__ == "__main__":
    config_file_path = "运维/ssh/config/config.yaml"
    abs_path = os.path.abspath(config_file_path)
    print(abs_path)

    # config_data = read_config_file(config_file_path)
    with open(config_file_path, "r") as yamlfile:
        config = yaml.safe_load(yamlfile)
    main(config)
