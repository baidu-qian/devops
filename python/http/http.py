'''
Author: magician
Date: 2024-08-07 23:47:19
LastEditors: magician
LastEditTime: 2024-08-07 23:53:33
FilePath: /python/运维/http/weixin-http.py
Description: 

Copyright (c) 2024 by ${git_name_email}, All Rights Reserved. 
'''
from flask import Flask,jsonify
import socket
import datetime

app = Flask(__name__)

def get_server_info():
    hostname = socket.gethostname()
    ip_address = socket.gethostbyname(hostname)
    current_time = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    return {
        "hostname": hostname,
        "ip_address": ip_address,
        "current_time": current_time
    }

@app.route('/status', methods=['GET'])
def server_info():
    info = get_server_info()
    return jsonify(info)

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080, debug=True)