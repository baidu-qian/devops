'''
Author: magician
Date: 2025-01-16 14:40:31
LastEditors: magician
LastEditTime: 2025-03-07 00:24:20
FilePath: /python/运维/disasterRecovery/disaterRecovery.py
Description: 

Copyright (c) 2025 by ${git_name_email}, All Rights Reserved. 
'''
import argparse
import logging
import os,yaml
import shutil
import platform
import subprocess
import re
import paramiko
from ruamel.yaml import YAML
import json
from io import StringIO
import xml.etree.ElementTree as ET
from io import BytesIO
from xml.dom import minidom
import toml
from typing import List


TABLE_FAMILY_MAP = {
    "admin_cleaner_start_cache": "ttl_family",
    "admin_config_sync": "info",
    "admin_dev_location_status": "status",
    "admin_dev_status": "dev_status",
    "admin_dev_threat_index_data": "data",
    "admin_fdti_data": "fdti",
    "admin_fdti_hxb_level": "fdti",
    "admin_last_start_message": "data",
    "admin_push_data_hxb_field_pool": "common_data,threat_env",
    "admin_threat_illegal_app_statistical_learning": "statistical_learning",
    "admin_threat_index": "data",
    "admin_threat_index_data": "data",
    "admin_threat_index_data_sync_status": "data",
    "admin_weekly_active_device": "status",
    "bb_apkinfo_cache": "content",
    "bb_ccb_push_service_cache": "security_event",
    "bb_dev_app_information_history": "information",
    "bb_dev_info_cache": "content",
    "bb_dev_info_current": "information",
    "bb_modem_pool_cache": "content",
    "bb_new_dev_fingerprint_factor_cache": "associate_id",
    "bb_security_event_data_track": "track",
    "bb_security_event_fact_id_track": "counter,fact_id",
    "bb_security_event_fact_risk": "",
    "bb_security_event_origin_record": "",
    "bb_start_commons_fields": "common_fields"
}

def update_json(json_data):
    """
    编辑json
    Args:
        json_data (json): json文件
    Returns:
        json: 编辑后的json
    """
    # 更新数据库
    if config["database"]["db"]["enable"]:
        for key, value in config["database"]["db"].items():
            if key in json_data["global"]["database"]["used"]:
                json_data["global"]["database"]["used"][key] = value
    # 添加rsync
    if config["database"]["transfer_rsync_es"]["enable"]:
        for key, value in config["database"]["transfer_rsync_es"].items():
            if key != "enable":
                if "transfer_rsync_es" not in json_data:
                    json_data["transfer_rsync_es"] = {}
                json_data["transfer_rsync_es"][key] = value
    # 更新redis
    if config["database"]["redis"]["enable"]:
        if config["database"]["redis"]["active_mode"] == "single":
            json_data["global"]["redis"]["used"]["active_mode"] = "single"
            for key, value in config["database"]["redis"]["single"].items():
                if key in json_data["global"]["redis"]["single"]:
                    json_data["global"]["redis"]["single"][key] = value
        if config["database"]["redis"]["active_mode"] == "cluster":
            json_data["global"]["redis"]["used"]["active_mode"] = "cluster"
            for key, value in config["database"]["redis"]["cluster"].items():
                if key in json_data["global"]["redis"]["cluster"]:
                    json_data["global"]["redis"]["cluster"][key] = value

    # 更新kibana
    for key, value in config["database"]["kibana"].items():
        if key in json_data["global"]["kibana"]:
            json_data["global"]["kibana"][key] = value
    login_info = json.dumps({"username": config["database"]["elasticsearch"]["username"], "password": config["database"]["elasticsearch"]["password"]})
    json_data["global"]["kibana"]["everisk.kibana.login"] = login_info


    # 更新elasticsearch
    if config["database"]["elasticsearch"]["enable"]:
        for key, value in config["database"]["elasticsearch"].items():
            if key in json_data["global"]["elasticsearch"]:
                json_data["global"]["elasticsearch"][key] = value
        json_data["global"]["elasticsearch"]["everisk.elasticsearch.login"] = config["database"]["elasticsearch"]["username"] + ":" + config["database"]["elasticsearch"]["password"]
    return json_data
def ssh_init_everisk_ha(ip,username,password,port,config):
    """
    设置指定应用的副本数

    Args:
        ip (str): IP 地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        config (dict): 配置

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
        with sftp.file("/home/"+username+"/app/init/config/config.json","r") as remote_file:
            json_data = json.load(remote_file)
        new_json_data = update_json(json_data)
        # 回写文件
        everisk_path = "/home/"+username+"/app/init/config/config.json"
        tmp_everisk_path = "/home/"+username+"/app/init/config/config.json.tmp"
        with sftp.file(tmp_everisk_path, 'w') as remote_file:
            remote_file.write(json.dumps(new_json_data, indent=2))
        sftp.rename(tmp_everisk_path,everisk_path)
        logging.info(f"更新config.json成功")
    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def local_init_everisk_ha(username,config):
    """
    设置指定应用的副本数

    Args:
        config (dict): 配置

    Returns:
        None: 无返回值
    """
    # 本地文件路径
    tmp_local_file_path = f"/home/{username}/app/init/config/config.json.tmp"
    local_file_path = f"/home/{username}/app/init/config/config.json"

    # 检查本地文件是否存在
    if not os.path.exists(local_file_path):
        print(f"本地文件 {local_file_path} 不存在，跳过更新。")
        return

    # 读取本地文件
    with open(local_file_path, "r") as local_file:
        json_data = json.load(local_file)

    # 更新 JSON 数据
    new_json_data = update_json(json_data)

    # 将更新后的 JSON 数据写回本地文件
    with open(tmp_local_file_path, "w") as local_file:
        # json.dump(new_json_data, local_file, indent=2)
        local_file.write(json.dumps(new_json_data, indent=2))
    shutil.move(tmp_local_file_path,local_file_path)
    print(f"本地文件 {local_file_path} 更新成功。")

def deploy_transfer_rsync(ip, username, password, port, file ):
    """部署transfer_rsync服务

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        file (str): 文件路径
    """
    transfer_dirctory = f'/home/{username}/app/transfer'
    transfer_rsync_es_directory = f'/home/{username}/app/transfer_rsync_es'
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)

        # 判断transfer目录是否存在
        command = f'test -d /data/{username}/app/transfer && echo "exists"'
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output == "exists":
            # 如果目录存在，执行 cp 命令
            ssh_client.exec_command(f'cp -r {transfer_dirctory} {transfer_rsync_es_directory}')
            # 上传本地文件到远程目录
            sftp = ssh_client.open_sftp()
            sftp.put(file, f'{transfer_rsync_es_directory}/config/application-remote.properties')  # 替换为目标文件名
            sftp.close()
            logging.info("文件已成功上传并备份。")
        else:
            logging.error(f"{transfer_dirctory}目录不存在，无法执行操作。")
    finally:
        # 关闭连接
        ssh_client.close()

def local_deploy_transfer_rsync(local_file_path, username):
    """部署transfer_rsync服务

    Args:
        local_file_path (str): 本地文件路径
        username (str): 用户名
    """
    transfer_directory = f'/home/{username}/app/transfer'
    transfer_rsync_es_directory = f'/home/{username}/app/transfer_rsync_es'
    
    # 判断transfer目录是否存在
    if os.path.isdir(transfer_directory):
        # 如果目录存在，执行相同逻辑

        # 复制目录到新的位置
        if not os.path.exists(transfer_rsync_es_directory):
            os.makedirs(transfer_rsync_es_directory)
        shutil.copytree(transfer_directory, transfer_rsync_es_directory, dirs_exist_ok=True)

        # 上传本地文件到目标目录
        target_file = os.path.join(transfer_rsync_es_directory, 'config', 'application-remote.properties')
        target_dir = os.path.dirname(target_file)

        # 确保目标目录存在
        if not os.path.exists(target_dir):
            os.makedirs(target_dir)

        # 复制文件到目标目录
        shutil.move(local_file_path, target_file)

        logging.info("文件已成功上传并备份。")
    else:
        logging.error(f"{target_file}目录不存在，无法执行操作。")

def update_docker_compose(ip,username,password,port):
    """更新docker-compose里面的内容 

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    yaml = YAML()
    yaml.indent(mapping=2, sequence=4, offset=2)  # 设置缩进风格
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        docker_compose_path = f"/home/{username}/app/transfer_rsync_es/bin/docker-compose.yml"
        tmp_docker_compose_path = f"/home/{username}/app/transfer_rsync_es/bin/docker-compose.yml.tmp"
        # 判断transfer目录是否存在
        command = f'test -d /home/{username}/app/transfer_rsync_es && echo "exists"'
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output == "exists":
            # 使用sftp传输文件
            sftp= ssh_client.open_sftp()
            with sftp.file(docker_compose_path,"r") as remote_file:
                compose_data = yaml.load(remote_file)
            # 更新指定服务的名称
            if "services" in compose_data and "transfer" in compose_data["services"]:
                # 获取当前服务的数据
                transfer_service = compose_data["services"].pop("transfer")
                # 将其添加为新的键
                compose_data["services"]["transfer_rsync_es"] = transfer_service
                compose_data["services"]["transfer_rsync_es"]["container_name"] = "transfer_rsync_es"
            for volume in compose_data["services"]["transfer_rsync_es"]["volumes"]:
                if isinstance(volume, dict) and volume.get("source") == f"/home/{username}/app/transfer/log":
                    volume["source"] = f"/home/{username}/app/transfer_rsync_es/log"  # 修改为新的值
                elif isinstance(volume, dict) and volume.get("source") == f"/home/{username}/data/transfer":
                    volume["source"] = f"/home/{username}/data/transfer_rsync_es"  # 修改为新的值
                elif isinstance(volume, dict) and volume.get("source") == f"/home/{username}/app/transfer/config/log4j2.xml":
                    volume["source"] = f"/home/{username}/app/transfer_rsync_es/config/log4j2.xml"  # 修改为新的值
            new_volume = {
                "type": "bind",
                "source": f"/home/{username}/app/transfer_rsync_es/config/application-remote.properties",
                "target": "/root/transfer/config/application-remote.properties",
                "read_only": True,
            }

            # 将新绑定配置添加到 volumes 列表中
            compose_data["services"]["transfer_rsync_es"]["volumes"].append(new_volume)            
            # 修改 ports 中的指定项
            if "services" in compose_data:
                service_name = "transfer_rsync_es"  # 替换为你的服务名称
                if service_name in compose_data["services"]:
                    ports = compose_data["services"][service_name].get("ports", [])
                    # 查找并修改特定的端口配置
                    for i, port_mapping in enumerate(ports):
                        if port_mapping == "15001:15001":  # 查找要修改的端口映射
                            ports[i] = "15101:15101"  # 替换为你想要的新端口映射，例如 "16001:15001"
                            break

            with sftp.file(tmp_docker_compose_path, 'w') as remote_file:
                yaml.dump(compose_data,remote_file)
            sftp.rename(tmp_docker_compose_path,docker_compose_path)
        else:
            logging.error("目录不存在，退出 SSH。")
    finally:
        # 关闭连接
        ssh_client.close()

def local_update_docker_compose(username):
    """更新docker-compose里面的内容 

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    yaml = YAML()
    yaml.indent(mapping=2, sequence=4, offset=2)  # 设置缩进风格
    docker_compose_path = f"/home/{username}/app/transfer_rsync_es/bin/docker-compose.yml"
    tmp_docker_compose_path = f"/home/{username}/app/transfer_rsync_es/bin/docker-compose.yml.tmp"
    transfer_rsync_es_path = os.path.dirname(docker_compose_path)
    # 判断 transfer_rsync_es 目录是否存在
    if os.path.isdir(transfer_rsync_es_path):
        # 读取 docker-compose.yml 文件
        with open(docker_compose_path, "r") as local_file:
            compose_data = yaml.load(local_file)
    # 更新指定服务的名称
        if "services" in compose_data and "transfer" in compose_data["services"]:
            # 获取当前服务的数据
            transfer_service = compose_data["services"].pop("transfer")
            # 将其添加为新的键
            compose_data["services"]["transfer_rsync_es"] = transfer_service
            compose_data["services"]["transfer_rsync_es"]["container_name"] = "transfer_rsync_es"
        for volume in compose_data["services"]["transfer_rsync_es"]["volumes"]:
            if isinstance(volume, dict) and volume.get("source") == f"/home/{username}/app/transfer/log":
                volume["source"] = f"/home/{username}/app/transfer_rsync_es/log"  # 修改为新的值
            elif isinstance(volume, dict) and volume.get("source") == f"/home/{username}/data/transfer":
                volume["source"] = f"/home/{username}/data/transfer_rsync_es"  # 修改为新的值
            elif isinstance(volume, dict) and volume.get("source") == f"/home/{username}/app/transfer/config/log4j2.xml":
                volume["source"] = f"/home/{username}/app/transfer_rsync_es/config/log4j2.xml"  # 修改为新的值
        new_volume = {
            "type": "bind",
            "source": f"/home/{username}/app/transfer_rsync_es/config/application-remote.properties",
            "target": "/root/transfer/config/application-remote.properties",
            "read_only": True,
        }
        # 将新绑定配置添加到 volumes 列表中
        compose_data["services"]["transfer_rsync_es"]["volumes"].append(new_volume)            
        # 修改 ports 中的指定项
        if "services" in compose_data:
            service_name = "transfer_rsync_es"  # 替换为你的服务名称
            if service_name in compose_data["services"]:
                ports = compose_data["services"][service_name].get("ports", [])
                # 查找并修改特定的端口配置
                for i, port_mapping in enumerate(ports):
                    if port_mapping == "15001:15001":  # 查找要修改的端口映射
                        ports[i] = "15101:15101"  # 替换为你想要的新端口映射，例如 "16001:15001"
                        break
        with open(tmp_docker_compose_path, 'w') as local_file:
            yaml.dump(compose_data,local_file)
        shutil.move(tmp_docker_compose_path,docker_compose_path)
        logging.info("transfer_rsync_es更新完成，文件已保存到本地。")
    else:
        logging.error("transfer_rsync_es目录不存在，退出脚本。")


def update_service_script(ip, username, password, port):
    """更新service.sh脚本中的transfer为transfer_rsync_es

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        
        # 定义脚本路径
        script_path = f"/home/{username}/app/transfer_rsync_es/bin/service.sh"
        tmp_script_path = f"{script_path}.tmp"
        
        # 读取脚本内容
        stdin, stdout, stderr = ssh_client.exec_command(f"cat {script_path}")
        content = stdout.read().decode()
        
        # 替换内容
        new_content = content.replace("transfer", "transfer_rsync_es")
        
        # 写入临时文件
        stdin, stdout, stderr = ssh_client.exec_command(f"echo '{new_content}' > {tmp_script_path}")
        
        # 替换原文件
        stdin, stdout, stderr = ssh_client.exec_command(f"mv {tmp_script_path} {script_path}")
        
        logging.info("service.sh脚本更新成功")
    except Exception as e:
        logging.error(f"更新service.sh脚本失败: {str(e)}")
    finally:
        # 关闭连接
        ssh_client.close()

def start_transfer_rsync_es(ip,username,password,port):
    """启动transfer_rsync_es

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 判断transfer目录是否存在
        command = f'test -d /data/{username}/app/transfer_rsync_es && echo "exists"'
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output == "exists":
            # 执行远程命令
            stdin, stdout, stderr = ssh_client.exec_command(f"cd /home/{username}/app/transfer_rsync_es/bin && docker-compose up -d")
            # 输出执行结果
            logging.info(stdout.read().decode())
        else:
            logging.error("transfer_rsync_es目录不存在，退出 SSH。")
    finally:
        # 关闭连接
        ssh_client.close()
    
def stop_transfer_rsync_es(ip,username,password,port):
    """停止transfer_rsync_es

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 判断transfer目录是否存在
        command = f'test -d /data/{username}/app/transfer_rsync_es && echo "exists"'
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output == "exists":
            # 执行远程命令
            stdin, stdout, stderr = ssh_client.exec_command(f"cd /home/{username}/app/transfer_rsync_es/bin && docker-compose down")
            # 输出执行结果
            logging.info(stdout.read().decode())
        else:
            logging.error("transfer_rsync_es目录不存在，退出 SSH。")
    finally:
        # 关闭连接
        ssh_client.close()

def local_start_transfer_rsync_es(username):
    """启动transfer_rsync_es

    Args:
        username (str): 用户名
    """
    transfer_rsync_es_path = f"/home/{username}/app/transfer_rsync_es"
    # 判断transfer目录是否存在
    if os.path.isdir(transfer_rsync_es_path):
        command = f"cd /home/{username}/app/transfer_rsync_es/bin && docker-compose up -d"
        process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = process.communicate()
        # 输出执行结果
        logging.info(stdout.read().decode())
    else:
        logging.error("transfer_rsync_es目录不存在，退出 SSH。")

def local_stop_transfer_rsync_es(username):
    """停止transfer_rsync_es

    Args:
        username (str): 用户名
    """
    transfer_rsync_es_path = f"/home/{username}/app/transfer_rsync_es"
    # 判断transfer目录是否存在
    if os.path.isdir(transfer_rsync_es_path):
        command = f"cd /home/{username}/app/transfer_rsync_es/bin && docker-compose down"
        process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = process.communicate()
        # 输出执行结果
        logging.info(stdout.read().decode())
    else:
        logging.error("transfer_rsync_es目录不存在，退出。")

def update_hbase_site(ip,username,password,port,hbase_file):
    """更新hbase-site.xml文件中的内容

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        hbase_file (str): hbase-site.xml文件路径
    """
    tmp_hbase_file = os.path.join(hbase_file,".tmp")
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 使用sftp传输文件
        sftp= ssh_client.open_sftp()
        with sftp.file(hbase_file,"r") as remote_file:
            # 检查文件是否为空
            remote_file.seek(0, 2)  # 移动到文件末尾
            if remote_file.tell() == 0:
                logging.error(f"{hbase_file}:该文件为空。")
            else:
                remote_file.seek(0)  # 移动回文件开头以便读取
            # 解析 XML 内容
            tree = ET.parse(remote_file)
            root = tree.getroot()
        # 查找 hbase.replication 属性
        replication_enabled = False
        for property in root.findall('property'):
            name = property.find('name')
            if name is not None and name.text == 'hbase.replication':
                replication_enabled = True
                value = property.find('value')
                if value is not None:
                    value.text = 'true'  # 修改值为 true
                break

        # 如果没有找到，则添加该属性
        if not replication_enabled:
            new_property = ET.Element('property')
            name_element = ET.SubElement(new_property, 'name')
            name_element.text = 'hbase.replication'
            value_element = ET.SubElement(new_property, 'value')
            value_element.text = 'true'
            root.append(new_property)
        # 将修改后的内容写入 BytesIO 对象
        xml_output = BytesIO()
        tree.write(xml_output, encoding='utf-8', xml_declaration=True)
        # 使用 minidom 格式化 XML 内容
        xml_output.seek(0)  # 重置指针到开始位置
        pretty_xml_str = minidom.parseString(xml_output.getvalue()).toprettyxml(indent="  ", newl="\n").strip()
        # 将字符串内容写回远程文件
        with sftp.file(tmp_hbase_file, "w") as remote_file:
            remote_file.write(pretty_xml_str.encode('utf-8'))  # 写入字节内
        sftp.rename(tmp_hbase_file,hbase_file)
        logging.info(f"{hbase_file}:写入完成")
    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def local_update_hbase_site(username,hbase_file):
    """更新hbase-site.xml文件中的内容

    Args:
        username (str): 用户名
        hbase_file (str): hbase-site.xml文件路径
    """
    tmp_hbase_file = os.path.json(hbase_file,".tmp")
    # 检查本地文件是否存在
    if not os.path.exists(hbase_file):
        logging.info(f"本地文件 {hbase_file} 不存在，跳过更新。")
        return

    # 读取本地文件
    with open(hbase_file, "r") as local_file:
        local_file.seek(0, 2)  # 移动到文件末尾
        if local_file.tell() == 0:
            logging.error(f"{hbase_file}:该文件为空。")
        else:
            local_file.seek(0)  # 移动回文件开头以便读取
        # 解析 XML 内容
        tree = ET.parse(local_file)
        root = tree.getroot()
    # 查找 hbase.replication 属性
    replication_enabled = False
    for property in root.findall('property'):
        name = property.find('name')
        if name is not None and name.text == 'hbase.replication':
            replication_enabled = True
            value = property.find('value')
            if value is not None:
                value.text = 'true'  # 修改值为 true
            break

    # 如果没有找到，则添加该属性
    if not replication_enabled:
        new_property = ET.Element('property')
        name_element = ET.SubElement(new_property, 'name')
        name_element.text = 'hbase.replication'
        value_element = ET.SubElement(new_property, 'value')
        value_element.text = 'true'
        root.append(new_property)
    # 将修改后的内容写入 BytesIO 对象
    xml_output = BytesIO()
    tree.write(xml_output, encoding='utf-8', xml_declaration=True)
    # 使用 minidom 格式化 XML 内容
    xml_output.seek(0)  # 重置指针到开始位置
    pretty_xml_str = minidom.parseString(xml_output.getvalue()).toprettyxml(indent="  ", newl="\n").strip()
    # 将字符串内容写回远程文件
    with open(tmp_hbase_file, "w") as local_file:
        local_file.write(pretty_xml_str.encode('utf-8'))  # 写入字节内
    shutil.move(tmp_hbase_file,hbase_file)
    logging.info(f"{hbase_file}:文件更新完成")

def check_hbase_replication(ip,username,password,port,remote_zookeeper):
    """ 检测hbase是否开启Replication，如未开启，则开启

    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        remote_zookeeper (str): zookeeper地址
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 判断
        command = f'echo list_peers | /home/{username}/server/hbase/hbase/bin/hbase shell  2> /dev/null |grep ENABLED'
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output:
            logging.info("已有hbase replication同步,跳过配置")
        else:
            logging.info("未检测到hbase replication,开始尝试配置同步....")
            command = f"echo \"add_peer '100', '{remote_zookeeper}'\" | /home/{username}/server/hbase/hbase/bin/hbase shell "
            stdin, stdout, stderr = ssh_client.exec_command(command)
            command = f'echo list_peers | /home/{username}/server/hbase/hbase/bin/hbase shell  2> /dev/null |grep ENABLED'
            stdin, stdout, stderr = ssh_client.exec_command(command)
            output = stdout.read().decode().strip()
            if output:
                logging.info("配置hbase replication成功!")
            else:
                logging.error("配置失败，脚本退出!")
                ssh_client.close()
                exit
    finally:
        # 关闭连接
        ssh_client.close()

def local_check_hbase_replication(username, remote_zookeeper):
    """ 检测hbase是否开启Replication，如未开启，则开启

    Args:
        username (str): 用户名
        remote_zookeeper (str): zookeeper地址
    """
    try:
        # 判断是否已开启 Replication
        command = f'echo list_peers | /home/{username}/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED'
        result = subprocess.run(command, shell=True, capture_output=True, text=True)
        output = result.stdout.strip()

        if output:
            logging.info("已有hbase replication同步, 跳过配置")
        else:
            logging.info("未检测到hbase replication, 开始尝试配置同步....")
            # 添加 Replication Peer
            command = f"echo \"add_peer '100', '{remote_zookeeper}'\" | /home/{username}/server/hbase/hbase/bin/hbase shell"
            subprocess.run(command, shell=True, capture_output=True, text=True)

            # 再次检查是否配置成功
            command = f'echo list_peers | /home/{username}/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED'
            result = subprocess.run(command, shell=True, capture_output=True, text=True)
            output = result.stdout.strip()

            if output:
                logging.info("配置hbase replication成功!")
            else:
                logging.error("配置失败，脚本退出!")
                exit(1)
    except Exception as e:
        logging.error(f"执行过程中发生错误: {e}")
        exit(1)

def check_hbase_status(ip,username,password,port):
    """ 检测hbase服务状态
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    Returns:
        [bool]: 是否正常
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        # 判断
        command = "echo status |  /home/" +username+ "/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep  dead| awk -F ',' '{print $4}'| tr -d ' dead'"
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        if output == "0":
            logging.info("hbase服务正常")
            return True
        else:
            logging.error("hbase服务异常，其中有 %s 状态异常", output)
            return False
    finally:
        # 关闭连接
        ssh_client.close()

def local_check_hbase_status(username):
    """ 检测hbase服务状态
    
    Args:
        username (str): 用户名
    Returns:
        [bool]: 是否正常
    """
    
    # 判断
    command = "echo status |  /home/" +username+ "/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep  dead| awk -F ',' '{print $4}'| tr -d ' dead'"
    result = subprocess.run(command, shell=True, capture_output=True, text=True)
    output = result.stdout.strip()
    if output == "0":
        logging.info("hbase服务正常")
        return True
    else:
        logging.error("hbase服务异常，其中有 %s 状态异常", output)
        return False
    
def check_replication_scope(ip,username,password,port):
    """ 检测hbase服务scope
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        for table, column_families in TABLE_FAMILY_MAP.items():
            for cf in column_families.split(','):
                # command = "echo \"describe '"+ table +"'\" | /home/" +username+ "/server/hbase/hbase/bin/hbase shell 2>/dev/null | grep -i \"REPLICATION_SCOPE\" | grep -i \""+cf+"\" | awk -F\"REPLICATION_SCOPE => \" '{print $2}' | tr -d \" \""
                command = f"echo \"describe '{table}'\" | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"
                stdin, stdout, stderr = ssh_client.exec_command(command)
                output = stdout.read().decode().strip()
                if output:
                    # 使用正则表达式提取REPLICATION_SCOPE的值
                    match = re.search(r"REPLICATION_SCOPE\s*=>\s*'(\d+)'", output)
                    if match:
                        replication_scope_value = match.group(1)
                        logging.info(f"table:{table},cf:{cf},REPLICATION_SCOPE:{replication_scope_value}")
                else:
                    logging.error("hbase服务异常")
    finally:
        # 关闭连接
        ssh_client.close()

def local_check_replication_scope(username):
    """ 检测hbase服务scope
    
    Args:
        username (str): 用户名
    """
    for table, column_families in TABLE_FAMILY_MAP.items():
        for cf in column_families.split(','):
            # command = "echo \"describe '"+ table +"'\" | /home/" +username+ "/server/hbase/hbase/bin/hbase shell 2>/dev/null | grep -i \"REPLICATION_SCOPE\" | grep -i \""+cf+"\" | awk -F\"REPLICATION_SCOPE => \" '{print $2}' | tr -d \" \""
            command = f"echo \"describe '{table}'\" | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"
            result = subprocess.run(command, shell=True, capture_output=True, text=True)
            output = result.stdout.strip()
            if output:
                # 使用正则表达式提取REPLICATION_SCOPE的值
                match = re.search(r"REPLICATION_SCOPE\s*=>\s*'(\d+)'", output)
                if match:
                    replication_scope_value = match.group(1)
                    logging.info(f"table:{table},cf:{cf},REPLICATION_SCOPE:{replication_scope_value}")
            else:
                logging.error("hbase服务异常")


def check_tables_existence(ip,username,password,port):
    """ 检测hbase表是否齐全
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    Returns:
        [bool]: 是否齐全
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        command = f"echo list | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"
        stdin, stdout, stderr = ssh_client.exec_command(command)
        output = stdout.read().decode().strip()
        # 遍历 TABLE_FAMILY_MAP 的键，检查是否存在于 table_list 中
        for key in TABLE_FAMILY_MAP.keys():
            if key in output:
                logging.info(f"{key} exists in the table list.")
            else:
                logging.error(f"{key} does not exist in the table list.")
                return False
        return True
    finally:
        # 关闭连接
        ssh_client.close()


def local_check_tables_existence(username):
    """ 检测hbase表是否齐全
    
    Args:
        username (str): 用户名
    Returns:
        [bool]: 是否齐全
    """
    command = f"echo list | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"
    result = subprocess.run(command, shell=True, capture_output=True, text=True)
    output = result.read().decode().strip()
    # 遍历 TABLE_FAMILY_MAP 的键，检查是否存在于 table_list 中
    for key in TABLE_FAMILY_MAP.keys():
        if key in output:
            logging.info(f"{key} exists in the table list.")
        else:
            logging.error(f"{key} does not exist in the table list.")
            return False
    return True

def set_tables_family(ip,username,password,port,value):
    """ 修改表的所有业务列族的REPLICATION_SCOPE值,1为打开表同步，0为关闭表同步
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
        value (str): REPLICATION_SCOPE值
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        for table, column_families in TABLE_FAMILY_MAP.items():


            for cf in column_families.split(','):
                # command = f"""echo "disable '{table}' | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"""
                # stdin, stdout, stderr = ssh_client.exec_command(command)
                # command = f"""echo "alter '{table}', {{NAME => '{cf}', REPLICATION_SCOPE => '{value}'}}" | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"""
                # # command = "echo \"alter '"+ table +"', {{NAME => '"+ cf +"', REPLICATION_SCOPE => '"+ value +"'}} \" | /home/"+ username +"/server/hbase/hbase/bin/hbase shell 2>/dev/null"
                # stdin, stdout, stderr = ssh_client.exec_command(command)
                # # output = stdout.read().decode().strip()
                # # print(output)
                # command = f"""echo "enable '{table}' | /home/{username}/server/hbase/hbase/bin/hbase shell 2>/dev/null"""
                # stdin, stdout, stderr = ssh_client.exec_command(command)
                # 定义 HBase shell 的路径
                hbase_shell_path = f"/home/{username}/server/hbase/hbase/bin/hbase shell"

                # 构建要执行的命令
                commands = f"""
                disable '{table}';
                alter '{table}', {{NAME => '{cf}', REPLICATION_SCOPE => '{value}'}};
                enable '{table}';
                """

                # 使用 SSH 执行命令
                stdin, stdout, stderr = ssh_client.exec_command(f"echo \"{commands}\" | {hbase_shell_path} 2>/dev/null")

                # 读取输出和错误
                output = stdout.read().decode().strip()
                error = stderr.read().decode().strip()

                # 打印输出和错误信息
                if output:
                    logging.info("Output:", output)
                if error:
                    logging.error("Error:", error)
    finally:
        # 关闭连接
        ssh_client.close()


def local_set_tables_family(username,value):
    """ 修改表的所有业务列族的REPLICATION_SCOPE值,1为打开表同步，0为关闭表同步
    
    Args:
        username (str): 用户名
        value (str): REPLICATION_SCOPE值
    """
    for table, column_families in TABLE_FAMILY_MAP.items():
        for cf in column_families.split(','):
            # 定义 HBase shell 的路径
            hbase_shell_path = f"/home/{username}/server/hbase/hbase/bin/hbase shell"

            # 构建要执行的命令
            commands = f"""
            disable '{table}';
            alter '{table}', {{NAME => '{cf}', REPLICATION_SCOPE => '{value}'}};
            enable '{table}';
            """

            # 使用 SSH 执行命令
            result = subprocess.run(f"echo \"{commands}\" | {hbase_shell_path} 2>/dev/null", shell=True, capture_output=True, text=True)
            output = result.read().decode().strip()

            # 打印输出和错误信息
            if output:
                logging.info("Output:", output)


def get_redis_config(redis_config, mode):
    """根据模式获取 Redis 配置"""
    if mode == "single":
        return {
            "cluster": False,
            "address": f"{redis_config['single']['spring.redis.host']}:{redis_config['single']['spring.redis.port']}",
            "password": redis_config['single']['spring.redis.password']
        }
    elif mode == "cluster":
        return {
            "cluster": True,
            "address": redis_config['cluster']['spring.redis.cluster.nodes'],
            "password": redis_config['cluster']['spring.redis.password']
        }
    else:
        raise ValueError(f"Unsupported active_mode: {mode}")

def edit_shake(config):
    """修改 redis-shake 的配置

    Args:
        config (dict): 配置
    """
    try:
        # 获取源 Redis 配置
        sync_reader_config = get_redis_config(config["database"]["redis"], config["database"]["redis"]["active_mode"])
        
        # 获取备份 Redis 配置
        redis_writer_config = get_redis_config(config["database"]["backup_redis"], config["database"]["backup_redis"]["active_mode"])

        # 修改 redis-shake 的配置
        script_dir = os.path.dirname(os.path.abspath(__file__))
        toml_file_path = os.path.join(script_dir, "file/redis/config/shake.toml")

        # 读取 TOML 文件
        with open(toml_file_path, "r") as f:
            toml_config = toml.load(f)

        # 更新配置
        toml_config["sync_reader"] = sync_reader_config
        toml_config["redis_writer"] = redis_writer_config

        # 写回 TOML 文件
        with open(toml_file_path, "w") as f:
            toml.dump(toml_config, f)

    except KeyError as e:
        raise ValueError(f"Missing required configuration key: {e}")
    except FileNotFoundError:
        raise FileNotFoundError(f"TOML file not found at path: {toml_file_path}")
    except PermissionError:
        raise PermissionError(f"No permission to read/write file at path: {toml_file_path}")
    except Exception as e:
        raise RuntimeError(f"An error occurred while updating redis-shake config: {e}")
    

def deploy_redis_shake(ip,username,password,port):
    """ 部署redis_shake服务
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        cpu_architecture = platform.machine()
        if cpu_architecture == "x86_64":
            redis_shake_path = "redis-shake-v4.3.2-linux-amd64.tar.gz"
        elif cpu_architecture == "aarch64":
            redis_shake_path = "redis-shake-v4.3.2-linux-arm64.tar.gz"
        else:
            raise ValueError("Unsupported CPU architecture")
        commands = f"""
            mkdir -p /home/{username}/ops/redis-shake/bin
            mkdir -p /home/{username}/ops/redis-shake/logs
            mkdir -p /home/{username}/ops/redis-shake/config
        """
        ssh_client.exec_command(commands)
        # 上传文件
        script_dir = os.path.dirname(os.path.abspath(__file__))
        redis_shake_file = os.path.join(script_dir,"file/redis/"+redis_shake_path)
        redis_service_file = os.path.join(script_dir,"file/redis/config/service.sh")
        redis_toml_file = os.path.join(script_dir,"file/redis/config/shake.toml")
        sftp = ssh_client.open_sftp()
        sftp.put(redis_shake_file, "/home/"+username+"/ops/redis-shake/bin/"+redis_shake_path)
        sftp.put(redis_service_file, "/home/"+username+"/ops/redis-shake/bin/service.sh")
        sftp.put(redis_toml_file, "/home/"+username+"/ops/redis-shake/config/shake.toml")
        sftp.close()
        # 执行命令
        commands = f"""
            cd /home/{username}/ops/redis-shake/bin 
            tar zxf {redis_shake_path} 
            rm -f {redis_shake_path} 
            chmod +x redis-shake 
            rm -f shake.toml 
            chmod +x ./service.sh 
            ./service.sh nodogstart
        """
        stdin, stdout, stderr = ssh_client.exec_command(commands)
        output = stdout.read().decode().strip()
        logging.info(output)
    finally:
        # 关闭连接
        sftp.close()
        ssh_client.close()

def remote_stop_redis_shake(ip,username,password,port):
    """ 停止redis_shake服务
    
    Args:
        ip (str): ip地址
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    # 创建ssh客户端
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        # 连接远程服务器
        ssh_client.connect(hostname=ip, username=username, password=password, port=port)
        commands = f"""
            cd /home/{username}/ops/redis-shake/bin
            ./service.sh stop
        """
        stdin, stdout, stderr = ssh_client.exec_command(commands)
        output = stdout.read().decode().strip()
        logging.info(output)
    finally:
        # 关闭连接
        ssh_client.close()


def local_deploy_redis_shake(username):
    """ 部署redis_shake服务
    
    Args:
        username (str): 用户名
    """
    
    cpu_architecture = platform.machine()
    if cpu_architecture == "x86_64":
        redis_shake_path = "redis-shake-v4.3.2-linux-amd64.tar.gz"
    elif cpu_architecture == "aarch64":
        redis_shake_path = "redis-shake-v4.3.2-linux-arm64.tar.gz"
    else:
        raise ValueError("Unsupported CPU architecture")
    commands = f"""
        mkdir -p /home/{username}/ops/redis-shake/bin
        mkdir -p /home/{username}/ops/redis-shake/logs
        mkdir -p /home/{username}/ops/redis-shake/config
    """
    subprocess.run(commands, shell=True, capture_output=True, text=True)
    # 上传文件
    script_dir = os.path.dirname(os.path.abspath(__file__))
    redis_shake_file = os.path.join(script_dir,"file/redis/"+redis_shake_path)
    redis_service_file = os.path.join(script_dir,"file/redis/config/service.sh")
    redis_toml_file = os.path.join(script_dir,"file/redis/config/shake.toml")
    shutil.copy(redis_shake_file, "/home/"+username+"/ops/redis-shake/bin/"+redis_shake_path)
    shutil.copy(redis_service_file, "/home/"+username+"/ops/redis-shake/bin/service.sh")
    shutil.copy(redis_toml_file, "/home/"+username+"/ops/redis-shake/config/shake.toml")
    # 执行命令
    commands = f"""
        cd /home/{username}/ops/redis-shake/bin 
        tar zxf {redis_shake_path} 
        rm -f {redis_shake_path} 
        chmod +x redis-shake 
        rm -f shake.toml 
        chmod +x ./service.sh 
        ./service.sh nodogstart
    """
    result = subprocess.run(commands, shell=True, capture_output=True, text=True)
    output = result.read().decode().strip()
    logging.info(output)
    
def local_stop_redis_shake(username):
    """本地停止redis-shake
    
    Args:
        username (str): 用户名
    """
    logging("本地停止redis-shake")
    commands = f"""
        cd /home/{username}/ops/redis-shake/bin
        ./service.sh stop
    """
    result = subprocess.run(commands, shell=True, capture_output=True, text=True)
    output = result.read().decode().strip()
    logging.info(output)

def remote_edit_hosts(ip, username, password, port=22):
    # 配置 SSH 客户端
    ssh_config = paramiko.SSHClient()
    ssh_config.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 连接远程服务器
        ssh_config.connect(ip, port=port, username=username, password=password)
    except Exception as e:
        logging.error(f"Failed to connect to remote server: {e}")
        return e

    # 需要注释的域名列表
    domains_to_comment = [
        "nginx", "redis", "crash", "postgres", "zookeeper", "hbase", "kafka",
        "elasticsearchMaster", "elasticsearchClient", "nebula", "kibana", "init",
        "receiver", "cleaner", "transfer", "threat", "web-service", "analyzer-dev",
        "security-event", "app-sender",
    ]

    # 构造 sed 命令来注释这些域名
    sed_command = "sudo sed -i "
    for domain in domains_to_comment:
        sed_command += f"-e '/{domain}/s/^/# /' "
    sed_command += "/etc/hosts"

    try:
        # 执行 sed 命令
        stdin, stdout, stderr = ssh_config.exec_command(sed_command)
        exit_status = stdout.channel.recv_exit_status()  # 获取命令执行状态
        if exit_status != 0:
            logging.error(f"Failed to run sed command: {stderr.read().decode()}")
            return stderr.read().decode()
    except Exception as e:
        logging.error(f"Failed to execute command: {e}")
        return e
    finally:
        ssh_config.close()

    # 输出结果
    logging.info("Command executed successfully.")
    return None

def read_lines(file_path: str) -> List[str]:
    """读取本地文件内容，返回行列表（去除末尾换行符）"""
    try:
        with open(file_path, 'r') as f:
            return [line.rstrip('\n') for line in f.readlines()]
    except Exception as e:
        logging.error(f"Failed to read hosts_path file: {e}")
        raise

def read_remote_file(ssh_client: paramiko.SSHClient, remote_path: str) -> List[str]:
    """读取远程文件内容，返回行列表（去除末尾换行符）"""
    try:
        stdin, stdout, stderr = ssh_client.exec_command(f"cat {remote_path}")
        exit_status = stdout.channel.recv_exit_status()
        if exit_status != 0:
            error = stderr.read().decode().strip()
            logging.error(f"Failed to read remote file: {error}")
            raise Exception(error)
        return [line.rstrip('\n') for line in stdout.read().decode().splitlines()]
    except Exception as e:
        logging.error(f"Failed to read remote file: {e}")
        raise

def remote_append_missing_lines(
    ssh_client: paramiko.SSHClient,
    lines_to_add: List[str],
    existing_lines: List[str]
) -> None:
    """逐行检查并追加缺失内容到远程文件"""
    for line in lines_to_add:
        line_clean = line.strip()
        if not line_clean:  # 跳过空行
            continue

        if line_clean not in existing_lines:
            # 使用双引号和转义处理特殊字符
            command = f"echo '{line}' | sudo tee -a /etc/hosts >/dev/null"
            stdin, stdout, stderr = ssh_client.exec_command(command)
            exit_status = stdout.channel.recv_exit_status()

            if exit_status != 0:
                error = stderr.read().decode().strip()
                raise Exception(f"Append failed: {error}")
            logging.info(f"Added line: {line}")
        else:
            logging.info(f"Line already exists, skipping: {line}")

def remote_add_hosts(ip: str, username: str, password: str, hosts_path: str, port: int = 22) -> None:
    """主函数：连接服务器并更新 hosts 文件"""
    # 配置 SSH 客户端
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        # 建立 SSH 连接
        ssh.connect(ip, port=port, username=username, password=password)
        logging.info("SSH connection established")

        # 读取本地文件
        lines_to_add = read_lines(hosts_path)
        logging.debug(f"Read {len(lines_to_add)} lines from local file")

        # 读取远程文件
        existing_lines = read_remote_file(ssh, "/etc/hosts")
        logging.debug(f"Read {len(existing_lines)} lines from remote file")

        # 更新远程文件
        remote_append_missing_lines(ssh, lines_to_add, existing_lines)
        logging.info("Remote /etc/hosts updated successfully")

    except Exception as e:
        logging.error(f"Operation failed: {str(e)}")
        raise  # 可根据需要改为 return str(e)
    finally:
        ssh.close()
        logging.info("SSH connection closed")

def local_edit_hosts() -> None:
    """注释本地 /etc/hosts 中的特定域名"""
    domains_to_comment = [
        "nginx", "redis", "crash", "postgres", "zookeeper", "hbase", "kafka",
        "elasticsearchMaster", "elasticsearchClient", "nebula", "kibana", "init",
        "receiver", "cleaner", "transfer", "threat", "web-service", "analyzer-dev",
        "security-event", "app-sender",
    ]

    # 构建 sed 命令
    sed_command = "sudo sed -i "
    for domain in domains_to_comment:
        sed_command += f"-e '/{domain}/s/^/# /' "
    sed_command += "/etc/hosts"

    try:
        result = subprocess.run(
            sed_command,
            shell=True,
            check=True,
            capture_output=True,
            text=True
        )
        logging.info(f"Command executed successfully. Output:\n{result.stdout}")
    except subprocess.CalledProcessError as e:
        logging.error(f"Failed to run sed command: {e}\nError output:\n{e.stderr}")
        raise

def read_lines(file_path: str) -> List[str]:
    """读取文件内容并返回行列表（保留换行符）"""
    try:
        with open(file_path, 'r') as f:
            return [line.rstrip('\n') for line in f]
    except Exception as e:
        logging.error(f"Failed to read file {file_path}: {e}")
        raise

def append_missing_lines(etc_hosts_path: str, lines_to_add: List[str], existing_lines: List[str]) -> None:
    """追加缺失的行到 hosts 文件"""
    try:
        # 需要 sudo 权限写入
        with open(etc_hosts_path, 'a') as f:
            for line in lines_to_add:
                line_clean = line.strip()
                if not line_clean:  # 跳过空行
                    continue
                
                if not contains(existing_lines, line_clean):
                    f.write(f"{line_clean}\n")
                    logging.info(f"Added line: {line_clean}")
                else:
                    logging.info(f"Line already exists, skipping: {line_clean}")
    except PermissionError:
        logging.error("Permission denied. Try running with sudo.")
        raise
    except Exception as e:
        logging.error(f"Failed to write to {etc_hosts_path}: {e}")
        raise

def contains(lines: List[str], target: str) -> bool:
    """检查是否包含目标行（忽略前后空白）"""
    return any(line.strip() == target.strip() for line in lines)       

def deploy_db_ha(ip,username,password,port):
    """部署数据库高可用 
    
    Args:
        ip (str): 服务器ip
        username (str): 用户名
        password (str): 密码
        port (int): 端口
    """
    logging("部署数据库高可用")

def setup_logger(local_file_path):
    """
    设置日志记录器，包括终端输出和文件输出。
    :param local_file_path: 日志文件存放的路径
    :return: 配置好的 logger 对象
    """
    logger = logging.getLogger()
    logger.setLevel(logging.INFO)  # 设置日志级别为 INFO

    # 创建格式化器
    formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')

    # 创建终端处理器（StreamHandler）
    console_handler = logging.StreamHandler()
    console_handler.setFormatter(formatter)

    # 创建文件处理器（FileHandler）
    log_file_path = os.path.join(os.path.dirname(local_file_path), "update_everisk.log")
    file_handler = logging.FileHandler(log_file_path, mode='a')  # 'a' 表示追加模式
    file_handler.setFormatter(formatter)

    # 将处理器添加到日志记录器
    logger.addHandler(console_handler)
    logger.addHandler(file_handler)

    return logger

def main(config):
    script_dir = os.path.dirname(os.path.abspath(__file__))
    # 设置日志记录器
    logger = setup_logger(script_dir)
    username = config["database"]["ssh"]["username"]
    password = config["database"]["ssh"]["password"]
    port = config["database"]["ssh"]["port"]
    # 创建 ArgumentParser 对象
    parser = argparse.ArgumentParser(description="威胁感知更新配置，实现威胁灾备功能.")

    # 添加 --update_init 参数
    parser.add_argument('--update_init', action='store_true', help='根据config.yam配置中的内容,更新config.json中的配置')

    # 添加 --update_transfer 参数
    parser.add_argument('--update_transfer', action='store_true', help='根据config.yam配置中的内容,复制~/app/transfer到~/app/transfer_rsync_es，并自动配置docker-compose.yaml的内容 及配置文件 ，然后启动')

    # 添加 --update_hbase 参数
    parser.add_argument('--update_hbase', action='store_true', help='根据config.yam配置中的内容,配置hbase的hbase.replication,然后自动配置hbase中的scope,并推送至远程服务器')

    # 添加 --update_redis 参数
    parser.add_argument('--update_redis', action='store_true', help='根据config.yam配置中的内容,启动一个redis_shake,将主的redis实时同步至血的redis')
    parser.add_argument('--stop_transferRsyncEs', action='store_true', help='根据config.yam配置中的内容,停止transfer_rsync_es服务')
    parser.add_argument('--stop_redisShake', action='store_true', help='根据config.yam配置中的内容,停止redis_shake')
    parser.add_argument('--stop_hbaseScope', action='store_true', help='根据config.yam配置中的内容,自动配置hbase中的scope为0')
    parser.add_argument('--edit_hosts', action='store_true', help='需要用root权限，或者sudo免密码权限，将template中的hosts追加到/etc/hosts中，此操作要在更新hbase之前完成')
    # 解析命令行参数
    args = parser.parse_args()
    islocal = config["database"]["local"]["islocal"]
    if islocal == False:
        # 根据参数执行相应的模块
        if args.update_init:
            for item in  config["database"]["init"]:
                if item["enable"]:
                    ip = item["ip"]
                    ssh_init_everisk_ha(ip,username,password,port,config)
        elif args.update_transfer:
            for item in config["database"]["transfer_rsync_es_list"]:
                if item["enable"]:
                    ip = item["ip"]
                    file = os.path.join(script_dir,"template/transfer/application-remote.properties")
                    deploy_transfer_rsync(ip,username,password,port,file)
                    update_docker_compose(ip,username,password,port)
                    update_service_script(ip,username,password,port)
                    start_transfer_rsync_es(ip,username,password,port)
        elif args.update_hbase:
            hbase_remote_file = f"/home/{username}/server/hbase/hbase/conf/hbase-site.xml"
            init_hbase_remote_file = f"/home/{username}/app/init/config/everisk/hadoop/hbase-site.xml"
            if config["database"]["hbase"]["enable"]:
                for ip in config["database"]["hbase"]["list"]:
                    update_hbase_site(ip,username,password,port,hbase_remote_file)
            for item in config["database"]["init"]:
                if item["enable"]:
                    ip = item["ip"]
                    update_hbase_site(ip,username,password,port,init_hbase_remote_file)
            if config["database"]["remote_zookeeper"]["enable"]:
                remote_zookeeper = config["database"]["remote_zookeeper"]["zookeeper.server"]
                hbase_ip = config["database"]["hbase"]["list"][0]
                check_hbase_replication(hbase_ip,username,password,port,f'{remote_zookeeper}:/hbase')
                hbase_status = check_hbase_status(hbase_ip,username,password,port)
                hbase_existence = check_tables_existence(hbase_ip,username,password,port)
                # if hbase_status and hbase_existence:
                if hbase_status:
                    check_replication_scope(hbase_ip,username,password,port)
                    # 开启hbase 副本推送
                    set_tables_family(hbase_ip,username,password,port,1)
                    check_replication_scope(hbase_ip,username,password,port)
        elif args.update_redis:
            edit_shake(config)
            if config["database"]["redis_sync"]["enable"]:
                ip = config["database"]["redis_sync"]["ip"]
                deploy_redis_shake(ip,username,password,port)
        elif args.stop_transferRsyncEs:
            for item in config["database"]["transfer_rsync_es_list"]:
                if item["enable"]:
                    ip = item["ip"]
                    stop_transfer_rsync_es(ip,username,password,port)
        elif args.stop_redisShake:
            ip = config["database"]["redis_sync"]["ip"]
            remote_stop_redis_shake(ip,username,password,port)
        elif args.stop_hbaseScope:
            hbase_status = check_hbase_status(hbase_ip,username,password,port)
            hbase_existence = check_tables_existence(hbase_ip,username,password,port)
            # if hbase_status and hbase_existence:
            if hbase_status:
                check_replication_scope(hbase_ip,username,password,port)
                # 开启hbase 副本推送
                set_tables_family(hbase_ip,username,password,port,0)
                check_replication_scope(hbase_ip,username,password,port)
        elif args.edit_hosts:
            # 获取用户确认
            user_input = input("请确认修改 template/hosts 内容 (yes/y 继续): ").strip().lower()
            if user_input not in ("yes", "y"):
                logging.error("未确认修改 template/hosts 内容，程序退出")
                return

            # 构建 hosts 文件路径
            current_dir = os.getcwd()
            hosts_path = os.path.join(current_dir, "template/hosts")
            # 关闭hbase 副本推送
            if config["database"]["hbase"]["enable"]:
                for ip in config["database"]["hbase"]["list"]:
                    remote_edit_hosts(ip,username,password,port)
                    remote_add_hosts(ip,username,password,hosts_path,port)
        else:
            # 如果没有提供任何参数，显示帮助信息
            parser.print_help()
    
    

if __name__ == "__main__":
    script_dir = os.path.dirname(os.path.abspath(__file__))
    config_file_path = os.path.join(script_dir, "config", "config.yaml")
    abs_path = os.path.abspath(config_file_path)
    # config_data = read_config_file(config_file_path)
    with open(config_file_path, "r") as yamlfile:
        config = yaml.safe_load(yamlfile)
    main(config)