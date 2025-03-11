# 使用方法
## 实现效果----自动格式化硬盘，并挂载到相应的目录 
1. 调整my_disk_hosts文件
```
# 将以下组里的ip改为所需要的挂载的硬盘
[my_disk]
172.16.36.150
172.16.36.185

#  调整远程用户名和密码

[all:vars]
ansible_ssh_port = 22 
ansible_ssh_user = admin
ansible_ssh_pass = 'admin' 
ansible_su_pass = 'admin' 
#host_key_checking = False 
```

2. 调整mount_disk.yml文件
```
# 重点调整disk_name和dir_path两个变量
  vars:
    vg_name: "admin_vg"
    lv_name: "admin_lv"
    disk_name: "/dev/sdb"
    dir_path: "/data"
    disk_num: 1


```

3. 挂载硬盘
```
ansible-playbook -i my_disk_hosts mount_disk.yml
```