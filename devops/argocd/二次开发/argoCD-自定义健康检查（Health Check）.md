# argoCD-自定义健康检查（Health Check）

## 前提条件

argoCD服务需要安装完成

[argoCD部署教程-20250327](../部署/argoCD/argoCD部署教程-20250327.md)

## 背景

argoCD可以通过Health Check定制开发，来完善美化原本没有的功能。

‍

## 部署

1. 准备一个lua脚本，我这边简单准备了一个

    ```shell
    -- 默认状态为进行中
    local status = "Progressing"
    local message = ""

    -- 确保status字段存在
    if obj.status ~= nil then
        -- 获取副本数配置
        local specReplicas = obj.spec.replicas or 1  -- 默认1个副本
        
        -- 检查就绪副本数
        local readyReplicas = obj.status.readyReplicas or 0
        local updatedReplicas = obj.status.updatedReplicas or 0
        
        -- 判断是否所有副本就绪
        if readyReplicas >= specReplicas then
            status = "Healthy"
            message = "所有副本就绪 (" .. readyReplicas .. "/" .. specReplicas .. ")"
        else
            -- 检查更新状态（滚动更新场景）
            if updatedReplicas < specReplicas then
                message = "正在滚动更新 (" .. updatedReplicas .. "/" .. specReplicas .. " 已更新)"
            else
                message = "等待副本就绪 (" .. readyReplicas .. "/" .. specReplicas .. ")"
            end
        end
        
        -- 检查Deployment是否卡住（例如镜像拉取失败）
        if obj.status.conditions ~= nil then
            for _, condition in ipairs(obj.status.conditions) do
                if condition.type == "Progressing" and condition.status == "False" then
                    status = "Degraded"
                    message = "部署停滞: " .. (condition.message or "未知原因")
                    break
                elseif condition.type == "Available" and condition.status == "False" then
                    status = "Degraded"
                    message = "可用性问题: " .. (condition.message or "未知原因")
                    break
                end
            end
        end
    else
        -- 无status字段的异常情况
        status = "Degraded"
        message = "无法获取部署状态"
    end

    return {
        status = status,
        message = message
    }
    ```

2. 修改 ArgoCD 的 ConfigMap：

    kubectl -n argocd edit configmap argocd-cm

    ```yaml
    # argocd-cm.yaml
    data:
      resource.customizations: |
        apps/Deployment:
          health.lua: |
            <上面的脚本内容>
    ```

3. 观察argoCD的web页面

    ![](http://viphtl.duckdns.org:15002/i/2025/03/28/67e58ec297b10.png)
    <img width="683" alt="image" src="https://github.com/user-attachments/assets/84f45dde-393d-44a6-9ce3-e7ab09089228" />

    已经变为了

    ![](http://viphtl.duckdns.org:15002/i/2025/03/28/67e58d5eef6c1.png)
    <img width="477" alt="image" src="https://github.com/user-attachments/assets/04ad7846-7ce2-4381-9fd9-c08f1a9847ea" />

 
    我们尝试删除一个pod

    ![](http://viphtl.duckdns.org:15002/i/2025/03/28/67e58d4c0a7bb.png)
    <img width="742" alt="image" src="https://github.com/user-attachments/assets/82263af0-8239-47d8-8633-418c450abfe0" />


    测试脚本是正常的

‍
