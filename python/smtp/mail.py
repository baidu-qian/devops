import smtplib
from email.mime.text import MIMEText

def send_email(subject, message, from_addr, to_addr, smtp_server='smtp.qq.com', port=465):
    # 创建邮件内容
    msg = MIMEText(message)
    msg['Subject'] = subject
    msg['From'] = from_addr
    msg['To'] = to_addr

    try: 
        # 如果使用 Gmail 作为 SMTP 服务器，需要开始 TLS 安全连接
        server = smtplib.SMTP_SSL(smtp_server, port,timeout=10)
        # 登录到 SMTP 服务器
        server.login(from_addr, 'xxxxxxxxx')  # 注意：这里的密码需要替换为实际的密码
        # 发送邮件
        server.send_message(msg)
        server.quit()
    except Exception as e:
        print(f"邮件发送失败: {str(e)}")
        try:
            # 如果 SSL 连接失败，尝试使用 TLS 连接
            server = smtplib.SMTP(smtp_server, port,timeout=10)
            server.starttls()  # 启用 TLS
            server.login(from_addr, 'xxxxxxxxxx')  # 替换为实际密码
            server.send_message(MIMEText(message))
            print("Email sent via TLS")
        except Exception as e:
            print(f"TLS connection failed: {e}")

# 示例使用
subject = "Hello from Python Script"
message = "This is a test email sent from a Python script."
from_addr = ""
to_addr = ""

send_email(subject, message, from_addr, to_addr)
