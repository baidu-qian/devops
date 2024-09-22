'''
Author: magician
Date: 2024-09-12 23:10:56
LastEditors: magician
LastEditTime: 2024-09-20 00:04:31
FilePath: /python/运维/服务降级/service-downgrade.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
import re
import paramiko
import os,yaml
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

def get_es_client_mem(ip,username,password,port):
    """
    获取es-client内存

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口

    Returns:
        tuple: 内存占用
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        
        # 使用sftp传输文件
        sftp = ssh_client.open_sftp()
        with sftp.file("/home/app/server/elasticsearchClient/elasticsearch/config/jvm.options", "r") as f:
            content = f.read().decode().splitlines()
        for line in content:
            if line.startswith("-Xms"):
                # xms = line.split("Xms")[1]
                xms = line[4:-1]
            elif line.startswith("-Xmx"):
                # xmx = line.split("Xmx")[1]
                xmx = line[4:-1]

        print(f"-Xms: {xms}")
        print(f"-Xmx: {xmx}")
            

    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def set_es_client_mem(ip,username,password,port,mem):
    """
    设置es-client内存

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        mem (int): 内存
    Returns:
        tuple: null
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)

        # 使用sftp传输文件
        sftp= ssh_client.open_sftp()
        with sftp.file("/home/app/server/elasticsearchClient/elasticsearch/config/jvm.options","r") as f:
            content = f.read().decode().splitlines()
        # 修改内存值
        for i, line in enumerate(content):
            if line.startswith("-Xms"):
                content[i] = f"-Xms{mem}g"  # 修改为新的值
            elif line.startswith("-Xmx"):
                content[i] = f"-Xmx{mem}g"  # 修改为新的值
        # 将修改后的内容写回文件
        with sftp.file("/home/app/server/elasticsearchClient/elasticsearch/config/jvm.options", "w") as f:
            f.write("\n".join(content).encode())

    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def server_stop(ip,username,password,port,server_name):
    """
    应用停止

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        server_name (str): 应用名称

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
        if server_name == "elasticsearch":
            stdin, stdout, stderr = ssh_client.exec_command("bash /home/app/server/elasticsearchClient/bin/service.sh stop")
            stdout.channel.recv_exit_status()  # 等待命令执行完成
    finally:
        # 关闭连接
        ssh_client.close()

def server_restart(ip, username, password, port, app_name):
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
        stdin, stdout, stderr = ssh_client.exec_command("bash /home/app/server/elasticsearchClient/bin/service.sh stop")
        stdout.channel.recv_exit_status()  # 等待命令执行完成
        # time.sleep(5)
        # print("停止信息:" + stdout)
        stdin, stdout, stderr = ssh_client.exec_command("bash /home/app/server/elasticsearchClient/bin/service.sh start")
        stdout.channel.recv_exit_status()  # 等待命令执行完成
        # print("启动信息:" + stdout)

    finally:
        ssh_client.close()

def app_stop(ip,username,password,port,app_name):
    """
    应用停止

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
        stdin, stdout, stderr = ssh_client.exec_command(f"bash /home/app/app/{app_name}/bin/service.sh stop")
        stdout.channel.recv_exit_status()  # 等待命令执行完成
    finally:
        # 关闭连接
        ssh_client.close()

def check_remote_file_exists(ip, username, password, port, file_path):
    '''
    检查远程文件是否存在

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        file_path (str): 文件路径

    Returns:
        bool: 是否存在
    '''
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 执行命令
        stdin, stdout, stderr = ssh_client.exec_command(f"test -e {file_path}")
        exit_status = stdout.channel.recv_exit_status()  # 等待命令执行完成
        return exit_status == 0
    except:
        return False
    finally:
        # 关闭连接
        ssh_client.close()

def find_files(ip,username,password,port,path):
    '''
    查找目录下的内存溢出文件

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        file_path (str): 文件路径

    Returns:
        bool: 是否存在
    '''
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 执行命令
        stdin, stdout, stderr = ssh_client.exec_command(f"find {path} -name hs_err*")
        hs_err_files = stdout.readlines()
        if len(hs_err_files) > 0:
            return True
        else:
            return False
    except:
        return False
    finally:
        # 关闭连接
        ssh_client.close()

def rm_files(ip,username,password,port,path):
    '''
    删除目录下的内存溢出文件

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        file_path (str): 文件路径

    Returns:
        bool: 是否删除
    '''
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 执行命令
        stdin, stdout, stderr = ssh_client.exec_command("find " + path + " -name hs_err* | xargs -i rm -f {}")
        exit_status = stdout.channel.recv_exit_status()  # 等待命令执行完成
        return exit_status == 0
    except:
        return False
    finally:
        # 关闭连接
        ssh_client.close()

# # 遍历配置字典
# ips = set()
# def extract_ips(data):
#     '''
#     遍历配置字典

#     Args:
#         data (dict): 配置字典

#     Returns:
#         list: 所有 IP 地址

#     '''
#     # 正则表达式匹配 IP 地址 (IPv4)
#     ip_pattern = re.compile(r'\b(?:\d{1,3}\.){3}\d{1,3}\b')
#     # 需要提取 IP 的模块列表
#     modules_to_extract = ["security-event", "threat", "analyzer-dev", "es"]
#     if isinstance(data, dict):
#         for key, value in data.items():
#             # 只处理我们关心的模块
#             if key in modules_to_extract:
#                 # 如果是目标模块，提取 IP
#                 if isinstance(value, list):
#                     for item in value:
#                         if 'ip' in item:
#                             ips.add(item['ip'])
#             else:
#                 extract_ips(value)
#     elif isinstance(data, list):
#         for item in data:
#             extract_ips(item)
#     # elif isinstance(data, str):
#     #     # 只提取纯 IP 地址，不包括带端口的
#     #     for match in ip_pattern.findall(data):
#     #         ips.add(match)
#     return ips

def main(config):
    username = config["database"]["ssh"]["username"]
    password = config["database"]["ssh"]["password"]
    port = config["database"]["ssh"]["port"]
    # ip_list =extract_ips(config)
    # for i in ip_list:
    #     print(i)
    for es_list in config["database"]["es"]:
        if es_list["enable"] == True:
            print(es_list["ip"])
            cpuinfo,meminfo,disk_output = get_remote_system_info(es_list["ip"], username, password, port)
            print(f"cpu信息：{cpuinfo},内存信息：{meminfo},磁盘信息：{disk_output}")
            file_exists = find_files(es_list["ip"], username, password, port, "/home/app/")

            if (float(cpuinfo) < 20 and int(meminfo) < 2) or file_exists:
                print("cpu或内存不足，需要降级")
                ## 降级es
                es_mem = get_es_client_mem(es_list["ip"], username, password, port)
                if es_mem > 4:
                    set_es_client_mem(es_list["ip"], username, password, port, int(es_mem)/2 ) 
                    server_restart(es_list["ip"], username, password, port, "es")
                else:
                    set_es_client_mem(es_list["ip"], username, password, port, 2)
                    server_restart(es_list["ip"], username, password, port, "es")
                ## 清理内存溢出文件
                rm_files(es_list["ip"], username, password, port, "/home/app/")
    for appsender_list in config["database"]["app-sender"]:
        if appsender_list["enable"] == True:
            print(appsender_list["ip"])
            cpuinfo,meminfo,disk_output = get_remote_system_info(appsender_list["ip"], username, password, port)
            print(f"cpu信息：{cpuinfo},内存信息：{meminfo},磁盘信息：{disk_output}")
            file_exists = find_files(appsender_list["ip"], username, password, port, "/home/app/")
            if (float(cpuinfo) < 20 and int(meminfo) < 2) or file_exists:
                print("cpu或内存不足，需要降级")
                if check_remote_file_exists(appsender_list["ip"], username, password, port, "/home/app/app/app-sender"):
                    app_stop(appsender_list["ip"], username, password, port, "app-sender")
                ## 清理内存溢出文件
                rm_files(appsender_list["ip"], username, password, port, "/home/app/")
    for securityevent_list in config["database"]["security-event"]:
        if securityevent_list["enable"] == True:
            print(securityevent_list["ip"])
            cpuinfo,meminfo,disk_output = get_remote_system_info(securityevent_list["ip"], username, password, port)
            print(f"cpu信息：{cpuinfo},内存信息：{meminfo},磁盘信息：{disk_output}")
            file_exists = find_files(securityevent_list["ip"], username, password, port, "/home/app/")
            if (float(cpuinfo) < 20 and int(meminfo) < 2) or file_exists:
                print("cpu或内存不足，需要降级")
                if check_remote_file_exists(securityevent_list["ip"], username, password, port, "/home/app/app/security-event"):
                    app_stop(securityevent_list["ip"], username, password, port, "security-event")
                ## 清理内存溢出文件
                rm_files(securityevent_list["ip"], username, password, port, "/home/app/")
    for threat_list in config["database"]["threat"]:
        if threat_list["enable"] == True:
            print(threat_list["ip"])
            cpuinfo,meminfo,disk_output = get_remote_system_info(threat_list["ip"], username, password, port)
            print(f"cpu信息：{cpuinfo},内存信息：{meminfo},磁盘信息：{disk_output}")
            file_exists = find_files(threat_list["ip"], username, password, port, "/home/app/")
            if (float(cpuinfo) < 20 and int(meminfo) < 2) or file_exists:
                print("cpu或内存不足，需要降级")
                if check_remote_file_exists(threat_list["ip"], username, password, port, "/home/app/app/threat"):
                    app_stop(threat_list["ip"], username, password, port, "threat")
                ## 清理内存溢出文件
                rm_files(threat_list["ip"], username, password, port, "/home/app/")

    for analyzer_dev_list in config["database"]["analyzer-dev"]:
        if analyzer_dev_list["enable"] == True:
            print(analyzer_dev_list["ip"])
            cpuinfo,meminfo,disk_output = get_remote_system_info(analyzer_dev_list["ip"], username, password, port)
            print(f"cpu信息：{cpuinfo},内存信息：{meminfo},磁盘信息：{disk_output}")
            file_exists = find_files(analyzer_dev_list["ip"], username, password, port, "/home/app/")
            if (float(cpuinfo) < 20 and int(meminfo) < 2) or file_exists:
                print("cpu或内存不足，需要降级")
                if check_remote_file_exists(analyzer_dev_list["ip"], username, password, port, "/home/app/app/analyzer-dev"):
                    app_stop(analyzer_dev_list["ip"], username, password, port, "analyzer-dev")
                ## 清理内存溢出文件
                rm_files(analyzer_dev_list["ip"], username, password, port, "/home/app/")
    

    
if __name__ == "__main__":
    config_file_path = "运维/服务降级/config/config.yaml"
    abs_path = os.path.abspath(config_file_path)
    print(abs_path)

    with open(config_file_path, "r") as yamlfile:
        config = yaml.safe_load(yamlfile)
    main(config)
