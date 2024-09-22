'''
Author: magician
Date: 2024-06-09 22:28:30
LastEditors: magician
LastEditTime: 2024-06-25 00:36:49
FilePath: /python/运维/postgres/select.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
import psycopg2

try:

    conn = psycopg2.connect(database="xxxx", user="xxxx", password="xxxx", host="xxxx", port="5432")

    # 获得游标对象
    cursor = conn.cursor()

    # 执行一个查询 
    cursor.execute("SELECT 1")

    # 获取查询结果
    result = cursor.fetchone()
    print(result)

    # 如果查询成功，则result 返回(1,)
    if result == (1,):
        print("查询成功")
    else:
        print("查询失败")

except psycopg2.Error as e:
    print("Unable to connect to the database")
    print(e)
