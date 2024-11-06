'''
Author: magician
Date: 2024-11-01 16:05:48
LastEditors: magician
LastEditTime: 2024-11-05 13:57:49
FilePath: /go/src/go_code/api替换/api.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''

import os,yaml
import re

def replace_in_files(directory):
    """
    查找目录中的 YAML 文件

    Args:
        directory (str): 文件路径

    Returns:
        []string: 文件路径list
    """
    yamls = []
    # 使用 os.walk 遍历目录及其子目录
    for root, dirs, files in os.walk(directory):
        for filename in files:
            if filename.endswith(('.yaml', '.yml')) and  not filename.endswith(('service.yaml')):
                file_path = os.path.join(root, filename)
                yamls.append(file_path)
            #with open(file_path, 'r', encoding='utf-8') as file:
            #    content = file.read()
            
            ## 替换所有{abcd}为baidu.com
            #new_content = content.replace('{abcd}', 'baidu.com')
            
            ## 将替换后的内容写回文件
            #with open(file_path, 'w', encoding='utf-8') as file:
            #    file.write(new_content)
    return yamls

def update_yaml_variable(file_path, variable_name, new_value):
    """
    更新 YAML 文件中指定变量的值

    Args:
        file_path (str): 文件路径
        variable_name (str): 变量名
        new_value (str): 新值

    Returns:
        None: 无返回值
    """
    try:
        # 读取 YAML 文件
        with open(file_path, 'r', encoding='utf-8') as file:
            data = yaml.load(file, Loader=yaml.FullLoader)

        # 更新指定变量的值
        if data['kind'] == "StatefulSet" or data['kind'] == "Deployment" or data['kind'] == "Job" :
            for container in data['spec']['template']['spec']['containers']:
                for env in container.get('env', []):
                    if env['name'] == variable_name:
                        env['value'] = str(new_value)

        # 将修改后的数据写回 YAML 文件
        with open(file_path, 'w', encoding='utf-8') as file:
            yaml.dump(data, file, default_flow_style=False, allow_unicode=True)

    except Exception as e:
        print(f"Error updating variable '{variable_name}' in YAML file '{file_path}': {e}")

def update_yaml_images(file_path, image_name):
    """
    更新 YAML 文件中镜像名

    Args:
        file_path (str): 文件路径
        image_name (str): 镜像名

    Returns:
        None: 无返回值
    """
    try:
        # 读取 YAML 文件
        with open(file_path, 'r', encoding='utf-8') as file:
            data = yaml.load(file, Loader=yaml.FullLoader)

        # 更新指定变量的值
        if data['kind'] == "StatefulSet" or data['kind'] == "Deployment" or data['kind'] == "Job" :
            for container in data['spec']['template']['spec']['containers']:
                image_parts = container['image'].rsplit(':', 1)
                if len(image_parts) == 2:
                    # Replace registry and keep tag intact
                    container['image'] = f"{image_name}/{image_parts[0].split('/')[-1]}:{image_parts[1]}"

        # 将修改后的数据写回 YAML 文件
        with open(file_path, 'w', encoding='utf-8') as file:
            yaml.dump(data, file, default_flow_style=False, allow_unicode=True)

    except Exception as e:
        print(f"Error updating variable '{image_name}' in YAML file '{file_path}': {e}")

def replace_line_with_placeholder(file_path, placeholder):
    """
    更新文件中nysql连接信息

    Args:
        file_path (str): 文件路径
        placeholder (str): mysql连接库

    Returns:
        None: 无返回值
    """ 
    with open(file_path, 'r') as file:
        lines = file.readlines()

    for i, line in enumerate(lines):
        if re.match(r'^\s*(?:db\.url\.0=jdbc:mysql|db\.url\.0=jdbc:postgresql)', line):
            lines[i] = placeholder + '\n'

    with open(file_path, 'w') as file:
        file.writelines(lines)


def main(config):
    kafka_brokers = config['database']['kafka']['brokers']
    redis_host = config['database']['redis']['host']
    redis_port = str(config['database']['redis']['port'])
    redis_str = redis_host + ":" + redis_port
    redis_password = config['database']['redis']['password']
    mysql_ip = config['database']['mysql']['ip']
    mysql_port = str(config['database']['mysql']['port'])
    mysql_username = config['database']['mysql']['username']
    mysql_password = config['database']['mysql']['password']
    mysql_url = f"{mysql_username}:{mysql_password}@tcp({mysql_ip}:{mysql_port})/api-sec?charset=utf8\&parseTime=True\&loc=Local"
    clickhouse_ip = config['database']['clickhouse']['ip']
    clickhouse_port = str(config['database']['clickhouse']['port'])
    clickhouse_username = config['database']['clickhouse']['username']
    clickhouse_password = config['database']['clickhouse']['password']
    clickhouse_url = f"clickhouse://{clickhouse_username}:{clickhouse_password}@{clickhouse_ip}:{clickhouse_port}/default"
    script_dir = os.path.dirname(os.path.abspath(__file__))
    current_year_directory = os.path.join(script_dir, "api-k8s-deploy")
    image_path = config['database']['image']['path']
    yamls = replace_in_files(current_year_directory)
    for i in yamls:
        update_yaml_variable(i, 'REDIS_ENDPOINTS', redis_str)
        update_yaml_variable(i, 'KAFKA_ENDPOINTS', kafka_brokers)
        update_yaml_variable(i, 'MYSQL_URL', mysql_url)
        update_yaml_variable(i, 'MYSQL_SERVICE_HOST', mysql_ip)
        update_yaml_variable(i, 'MYSQL_SERVICE_PORT', mysql_port)
        update_yaml_variable(i, 'MYSQL_SERVICE_USER', mysql_username)
        update_yaml_variable(i, 'MYSQL_SERVICE_PASSWORD', mysql_password)
        update_yaml_variable(i, 'MYSQL_ADDRESS', mysql_ip+":"+mysql_port)
        update_yaml_variable(i, 'MYSQL_USER', mysql_username)
        update_yaml_variable(i, 'MYSQL_PASSWORD', mysql_password)
        update_yaml_variable(i, 'CLICKHOUSE_URL', clickhouse_url)
        update_yaml_variable(i, 'CLICKHOUSE_ADDRESS', clickhouse_ip+":"+clickhouse_port)
        update_yaml_variable(i, 'CLICKHOUSE_USER', clickhouse_username)
        update_yaml_variable(i, 'CLICKHOUSE_PASSWORD', clickhouse_password)
        replace_line_with_placeholder(i, '    db.url.0=jdbc:mysql://'+mysql_ip+':'+mysql_port+'/nacos?${MYSQL_SERVICE_DB_PARAM:characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false&allowPublicKeyRetrieval=true}')
        update_yaml_images(i, image_path)

if __name__ == "__main__":
    script_dir = os.path.dirname(os.path.abspath(__file__))
    config_file_path = os.path.join(script_dir, "config", "config.yaml")
    abs_path = os.path.abspath(config_file_path)
    print(abs_path)

    # config_data = read_config_file(config_file_path)
    with open(config_file_path, "r") as yamlfile:
        config = yaml.safe_load(yamlfile)
    main(config)