# argoCD-使用钉钉机器人通知GitOps应用变更


## 前提条件

* argoCD服务需要安装完成

  [argoCD部署教程-20250327](argoCD部署教程-20250327.md)
* 钉钉新建一个webhook

  https://open.dingtalk.com/document/orgapp/assign-a-webhook-url-to-an-internal-chatbot

‍

## 背景

argoCD可以通过Webhook扩展，将状态信息通过钉钉/微信将argoCD的信息暴露出来

‍

## 部署

### 用python尝试发送

1. 先简单写个python脚本，测试一下钉钉是否可用

    ```python
    '''
    Descripttion: 
    Author: Magician
    version: 
    Date: 2025-03-28 17:54:24
    LastEditors: Magician
    LastEditTime: 2025-03-28 18:49:58
    '''
    import requests
    import json

    def send_dingtalk_message(webhook_url, content, at_mobiles=None, is_at_all=False):
        """
        发送钉钉机器人消息
        :param webhook_url: 钉钉机器人的Webhook地址
        :param content: 要发送的文本内容
        :param at_mobiles: 被@人的手机号列表（可选）
        :param is_at_all: 是否@所有人（默认False）
        :return: 钉钉API的响应结果
        """
        headers = {
            "Content-Type": "application/json",
            "Charset": "UTF-8"
        }
        
        data = {
            "msgtype": "markdown",
            "text": {
                "content": content
            },
            "markdown": {
                "title": "# magician监控报警",
                "text": content
            },
            "at": {
                "atMobiles": at_mobiles if at_mobiles else [],
                "isAtAll": is_at_all
            }
        }
        
        try:
            response = requests.post(
                url=webhook_url,
                headers=headers,
                data=json.dumps(data)
            )
            result = response.json()
            if response.status_code == 200:
                print("消息发送成功")
                return result
            else:
                print(f"消息发送失败，错误码：{response.status_code}，错误信息：{result}")
                return None
        except requests.exceptions.RequestException as e:
            print(f"请求异常：{str(e)}")
            return None

    # 使用示例
    if __name__ == "__main__":
        # 替换为你的钉钉机器人Webhook地址
        webhook_url = "https://oapi.dingtalk.com/robot/send?access_token=<你的token>"
        
        # 消息内容
        message_content = "magician监控报警：服务器CPU使用率超过90%！请及时处理！"
        
        # 指定要@的人（手机号列表），如果不需要@特定人则设置为None
        at_mobiles = ["xxxxx", "xxxxx"]
        
        # 发送消息（is_at_all=True表示@所有人）
        send_dingtalk_message(
            webhook_url=webhook_url,
            content=message_content,
            at_mobiles=at_mobiles,
            is_at_all=False
        )
    ```

2. 运行python后，钉钉正常收到消息，说明链路和token是正常的

### 调整argoCD

1. 编辑configmap

    ```shell
    kubectl edit cm argocd-notifications-cm -n argocd
    ```

2. 将如下yaml按模板写入到刚刚的终端中

    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: argocd-notifications-cm
      namespace: argocd
    data:
      service.webhook.dingtalk: |
        url: https://oapi.dingtalk.com/robot/send?access_token=<你的token>
        headers:
          - name: Content-Type
            value: application/json
      context: |
        argocdUrl: https://argocd-server.argocd.svc.cluster.local
      template.app-sync-change: |
        webhook:
          dingtalk:
            method: POST
            body: |
              {
                    "msgtype": "markdown",
                    "markdown": {
                        "title":"magician: ArgoCD应用状态",
                        "text": "#magician ### ArgoCD应用状态\n> - 应用名称: {{.app.metadata.name}}\n> - 同步状态: {{ .app.status.operationState.phase}}\n> - 时间: {{.app.status.operationState.finishedAt}}\n> - 应用URL: [点击跳转]({{.context.argocdUrl}}/applications/{{.app.metadata.name}}?operation=true) \n"
                    }
              }
      template.app-sync-status-unknown: |
        webhook:
          dingtalk:
            method: POST
            body: |
              {
                    "msgtype": "markdown",
                    "markdown": {
                        "title":"magician: ArgoCD应用Unknown",
                        "text": "#magician ### ArgoCD应用Unknown\n> - <font color=\"warning\">应用名称</font>: {{.app.metadata.name}}\n> - <font color=\"warning\">应用同步状态</font>: {{.app.status.sync.status}}\n> - <font color=\"warning\">应用健康状态</font>: {{.app.status.health.status}}\n> - <font color=\"warning\">时间</font>: {{.app.status.operationState.startedAt}}\n> - <font color=\"warning\">应用URL</font>: [点击跳转ArgoCD UI]({{.context.argocdUrl}}/applications/{{.app.metadata.name}}?operation=true)"
                    }
              }
      template.app-sync-failed: |
        webhook:
          dingtalk:
            method: POST
            body: |
              {
                    "msgtype": "markdown",
                    "markdown": {
                        "title":"magician:ArgoCD应用发布失败",
                        "text": "#magician: ### ArgoCD应用发布失败\n> - <font color=\"danger\">应用名称</font>: {{.app.metadata.name}}\n> - <font color=\"danger\">应用同步状态</font>: {{.app.status.operationState.phase}}\n> - <font color=\"danger\">应用健康状态</font>: {{.app.status.health.status}}\n> - <font color=\"danger\">时间</font>: {{.app.status.operationState.startedAt}}\n> - <font color=\"danger\">应用URL</font>: [点击跳转ArgoCD UI]({{.context.argocdUrl}}/applications/{{.app.metadata.name}}?operation=true)"
                    }
              }
      trigger.on-deployed: |
        - description: Application is synced and healthy. Triggered once per commit.
          oncePer: app.status.sync.revision
          send: [app-sync-change]
          # trigger condition
          when: app.status.operationState.phase in ['Succeeded'] and app.status.health.status == 'Healthy'
      trigger.on-health-degraded: |
        - description: Application has degraded
          send: [app-sync-change]
          when: app.status.health.status == 'Degraded'
      trigger.on-sync-failed: |
        - description: Application syncing has failed
          send: [app-sync-failed]
          when: app.status.operationState != nil and app.status.operationState.phase in ['Error',
            'Failed']
      trigger.on-sync-status-unknown: |
        - description: Application status is 'Unknown'
          send: [app-sync-status-unknown]
          when: app.status.sync.status == 'Unknown'
      trigger.on-sync-running: |
        - description: Application is being synced
          send: [app-sync-change]
          when: app.status.operationState != nil and app.status.operationState.phase in ['Running']
      trigger.on-sync-succeeded: |
        - description: Application syncing has succeeded
          send: [app-sync-change]
          when: app.status.operationState != nil and app.status.operationState.phase in ['Succeeded']
      subscriptions: |
        - recipients: [dingtalk]
          triggers: [on-sync-failed, on-sync-succeeded, on-sync-status-unknown,on-deployed]
    ```

3. 终端查看日志

    ```shell
    kubectl -n argocd logs -f argocd-notifications-controller-68459f6cbb-9vdwp
    ```

    日志没有异常

    ```shell
    ready sent to '{dingtalk }' using the configuration in namespace argocd" resource=argocd/hello
    time="2025-03-28T14:00:22Z" level=info msg="Trigger on-sync-failed result: [{[0].QocUZC1QzO--b2WEwALX85YPE5o  [app-sync-failed] false}]" resource=argocd/hello
    time="2025-03-28T14:00:22Z" level=info msg="Trigger on-sync-succeeded result: [{[0].IwKhHw9Hu8IE3z5Y8CQE4vReLYs  [app-sync-change] true}]" resource=argocd/hello
    ```

4. argoCD控制页面中，删除deploy
5. 发现告警信息成功推送到dingtalk
