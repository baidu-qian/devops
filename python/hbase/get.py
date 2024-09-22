import happybase

# HBase 连接配置
host = '172.16.44.155'  # 替换为实际的 HBase 主机名或 IP 地址
port = 61000  # 替换为实际的 HBase 端口号
table_name = 'hbase_1102'  # 替换为实际的表名

# 连接到 HBase
connection = happybase.Connection(host=host, port=port)

# 打开表
table = connection.table(table_name)

# 执行 Scan 操作
scan_result = table.scan()

# 指定要写入的文件路径
output_file = 'hbase_data.txt'

# 将扫描结果写入到本地文件
with open(output_file, 'w') as f:
    for key, data in scan_result:
        f.write(f"Key: {key}, Data: {data}\n")

# 关闭连接
connection.close()

print(f"数据已成功写入到文件 {output_file}")
