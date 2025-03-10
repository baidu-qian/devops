/*
 * @Author: magician
 * @Date: 2023-07-29 17:29:48
 * @LastEditors: magician
 * @LastEditTime: 2023-09-13 01:57:13
 * @FilePath: /go/src/go_code/zstack/auto-crate/auto.go
 * @Description:
 * Copyright (c) 2023 by magician, All Rights Reserved.
 */
package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

var api_url = "http://172.16.44.100:8080/zstack/v1"

func sha512_hash(password string) string {
	hash := sha512.New()
	hash.Write([]byte(password))
	hash_value := hash.Sum(nil)
	hash_string := hex.EncodeToString(hash_value)
	return hash_string
}

func api_get_call(url string, headers map[string]interface{}) (map[string]interface{}, error) {
	// 创建client, 超时5s
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value.(string))
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("请求错误:%d", resp.StatusCode)
	}
	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// post接口
func api_post_call(url string, headers map[string]interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	// 创建client, 超时5s
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json_data))
	if err != nil {
		return nil, err
	}
	// 设置请求头
	for k, v := range headers {
		req.Header.Set(k, v.(string))
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		errorMsg := ""
		errData := make(map[string]interface{})
		err = json.NewDecoder(resp.Body).Decode(&errData)
		if err == nil {
			errorMsg = errData["error"].(string)
		}

		fmt.Println("ERROR:", errorMsg)
		return nil, fmt.Errorf("failed to make an API call, %s, %s", resp.Status, resp.Status)
	}
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	location, ok := response["location"].(string)
	if !ok {
		return nil, errors.New("location is null")
	}
	return query_post_state(location, headers)
}

// put接口
func api_put_call(url string, headers map[string]interface{}, data map[string]interface{}) (string, error) {
	// 创建client, 超时5s
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	// 创建请求
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(json_data))
	if err != nil {
		return "", err
	}
	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value.(string))
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("请求错误:%d", resp.StatusCode)
	}
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	return response["inventory"].(map[string]interface{})["uuid"].(string), nil
}

// post_state 接口
func query_post_state(location string, headers map[string]interface{}, retryCount ...int) (map[string]interface{}, error) {
	// 延时2s，刚刚创建完，拿不到数据
	time.Sleep(2 * time.Second)
	// 给循环设置默认值
	retry_count := 0
	if len(retryCount) > 0 {
		retry_count = retryCount[0]
	}
	// 最多重试5次
	if retry_count > 5 {
		return nil, errors.New("retry count is over 5")
	}
	rsp, err := api_get_call(location, headers)
	if err != nil {
		fmt.Println("任务未准备完成...")
		time.Sleep(5 * time.Second)
		return query_post_state(location, headers, retry_count+1)
	}
	state, ok := rsp["inventory"].(map[string]interface{})["state"].(string)
	if !ok {
		return nil, errors.New("state is null")
	}
	if state == "Running" {
		return rsp, nil
	}
	fmt.Printf("创建任务状态: %s\n", state)
	time.Sleep(5 * time.Second)
	return query_post_state(location, headers, retry_count+1)
}

// 登录入口
func login(password string) (string, error) {
	url := api_url + "/accounts/login"
	headers_str := `{
        "Content-Type": "application/json;charset=UTF-8"
    }`
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	data_str := fmt.Sprintf(`{
        "logInByAccount": {
            "accountName": "admin",
            "password": "%s"
        }
    }`, password)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(data_str), &data)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	resp, err := api_put_call(url, headers, data)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return resp, nil
}

func query_vm(session_uuid string) {
	url := api_url + "/vm-instances"
	headers_str := fmt.Sprintf(`
	{
        "Content-Type": "application/json;charset=UTF-8",
        "Authorization": "OAuth %s"
    }`, session_uuid)
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
	}
	data, err := api_get_call(url, headers)
	inventories := data["inventories"].([]interface{})
	for _, inventory := range inventories {
		fmt.Println("主机信息为:", inventory.(map[string]interface{})["name"].(string))
	}
}

func query_instance_offering(session_uuid string, cpum string) (string, error) {
	url := api_url + "/instance-offerings"
	headers_str := fmt.Sprintf(`
	{
		"Content-Type": "application/json;charset=UTF-8",
		"Authorization": "OAuth %s"
	}`, session_uuid)
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
	}
	data, err := api_get_call(url, headers)
	instance_offerings := data["inventories"].([]interface{})
	for _, inventory := range instance_offerings {
		instance_offering := inventory.(map[string]interface{})
		if instance_offering["name"].(string) == cpum {
			return instance_offering["uuid"].(string), nil
		}
	}
	return "", err
}

// 查询镜像
func query_image(session_uuid string, image string) (string, error) {
	url := api_url + "/images"
	headers_str := fmt.Sprintf(`
	{
		"Content-Type": "application/json;charset=UTF-8",
		"Authorization": "OAuth %s"
	}`, session_uuid)
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
	}
	data, err := api_get_call(url, headers)
	images := data["inventories"].([]interface{})
	for _, inventory := range images {
		image_list := inventory.(map[string]interface{})
		if image_list["name"].(string) == image {
			return image_list["uuid"].(string), nil
		}
	}
	return "", err
}

// 查询l3网络
func query_l3(session_uuid string, l3 string) (string, error) {
	url := api_url + "/l3-networks"
	headers_str := fmt.Sprintf(`
	{
		"Content-Type": "application/json;charset=UTF-8",
		"Authorization": "OAuth %s"
	}`, session_uuid)
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
	}
	data, err := api_get_call(url, headers)
	l3s := data["inventories"].([]interface{})
	for _, inventory := range l3s {
		l3_list := inventory.(map[string]interface{})
		if l3_list["name"].(string) == l3 {
			return l3_list["uuid"].(string), nil
		}
	}
	return "", err
}

// 创建vm虚拟机
func create_vm(session_uuid string, vm_name string, cpum string, image string, l3 string) (string, error) {
	url := api_url + "/vm-instances"
	headers_str := fmt.Sprintf(`
	{
		"Content-Type": "application/json;charset=UTF-8",
		"Authorization": "OAuth %s"
	}`, session_uuid)
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	data_str := fmt.Sprintf(`
	{
        "params":{
            "name":"%s",
            "instanceOfferingUuid": "%s",
            "imageUuid": "%s",
            "l3NetworkUuids":[
                "%s" 
            ],
            "dataDiskOfferingUuids":[
                "c49bb91aeabc4614a3fdf1292dcc62e3"
            ],
            "description":"golang脚本自动创建的虚拟机，如有疑问请联系尤洪春",
            "strategy":"InstantStart"
        },
        "systemTags":[
        ],
        "userTags":[
        ]
    }`, vm_name, cpum, image, l3)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(data_str), &data)
	if err != nil {
		fmt.Println(err)
	}
	resp, err := api_post_call(url, headers, data)
	if err != nil {
		fmt.Println(err)
	}
	// var ip map[string]interface{}
	// err = json.Unmarshal([]byte(resp), &ip)
	// if err != nil {
	// fmt.Println(err)
	// }
	// fmt.Println(resp["inventory"])
	// fmt.Println(resp["inventory"].(map[string]interface{})["vmNics"])
	// fmt.Println(resp["inventory"].(map[string]interface{})["vmNics"].(map[string]interface{})[0])
	vm_Nics, ok := resp["inventory"].(map[string]interface{})["vmNics"]
	if !ok {
		fmt.Errorf("找不到vmNics")
	}
	for _, nic := range vm_Nics.([]interface{}) {
		nic_map, ok := nic.(map[string]interface{})
		if !ok {
			fmt.Errorf("nic_map打失败")
		}
		ip, ok := nic_map["ip"]
		if !ok {
			fmt.Errorf("ip打失败")
		}
		fmt.Println(ip)
	}
	return "", err
}

func get_harbor_tag(ip, repository, tag string) bool {
	url := fmt.Sprintf("http://%s/api/v2.0/projects/admin/repositories/%s/artifacts/%s/tags", ip, repository, tag)
	headers := map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return false
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("请求错误:", err)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return false
	}

	return true
}

// build and  push docker镜像
func build_and_push_docker(username, password, image_name, image_tag, dockerfile_path, remote_repository, image_path string) {
	// 判断镜像是否存在
	if get_harbor_tag(remote_repository, image_name, image_tag) {
		fmt.Println("镜像已存在")
		return
	}
	if image_name == "web-service-nginx" {
		// 构建镜像
		cmd := exec.Command("docker", "build", "-t", image_name+":"+image_tag, "-f", dockerfile_path, ".", "--no-cache")
		if err := cmd.Run(); err != nil {
			fmt.Println("构建镜像失败")
			return
		}
	} else {
		// 构建镜像
		cmd := exec.Command("docker", "build", "-t", image_name+":"+image_tag, "-f", dockerfile_path, "--build-arg", "app_name="+image_name, ".", "--no-cache")
		if err := cmd.Run(); err != nil {
			fmt.Println("构建镜像失败")
			return
		}
	}
	// 标记镜像为远程仓库地址
	tag_cmd := exec.Command("docker", "tag", image_name+":"+image_tag, remote_repository+"/admin/"+image_name+":"+image_tag)
	if err := tag_cmd.Run(); err != nil {
		fmt.Println("构建镜像失败")
		return
	}
	// 登陆远程harbor
	login_cmd := exec.Command("docker", "login", "-u", username, "-p", password, remote_repository)
	if err := login_cmd.Run(); err != nil {
		fmt.Println("登陆远程harbor失败")
		return
	}
	// 推送镜像至远程harbor
	push_cmd := exec.Command("docker", "push", remote_repository+"/admin/"+image_name+":"+image_tag)
	if err := push_cmd.Run(); err != nil {
		fmt.Println("推送镜像至远程harbor失败")
		return
	}
	// 清理本地镜像
	clean_cmd := exec.Command("docker", "rmi", image_name+":"+image_tag)
	if err := clean_cmd.Run(); err != nil {
		fmt.Println("清理镜像失败")
		return
	}
	clean_cmd = exec.Command("docker", "rmi", remote_repository+"/admin/"+image_name+":"+image_tag)
	if err := clean_cmd.Run(); err != nil {
		fmt.Println("清理镜像失败")
		return
	}
	if image_name == "web-service-nginx" {
		// 保存镜像到本地
		save_cmd := exec.Command("docker", "save", "-o", image_path+"/"+image_name+".tar", remote_repository+"/admin/"+image_name+":"+image_tag)
		if err := save_cmd.Run(); err != nil {
			fmt.Println("保存镜像失败")
			return
		}
	} else {
		// 保存镜像到本地
		save_cmd := exec.Command("docker", "save", "-o", image_path+"/"+image_name+".tar", remote_repository+"/admin/"+image_name+":"+image_tag)
		if err := save_cmd.Run(); err != nil {
			fmt.Println("保存镜像失败")
			return
		}
	}
	if image_name == "init" {
		// copy init
		copy_cmd := exec.Command("cp", "-rf", "init.tar.gz", image_path+"../init.tar.gz")
		if err := copy_cmd.Run(); err != nil {
			fmt.Println("复制init失败")
			return
		}
	}
	fmt.Println("Docker image save completed successfully!")
	fmt.Println("Docker image build and  push completed successfully!")

}

// 替换hosts-magician-deploy配置文件
func replace_ips(file_path, ip, magician_tag, ui_tag string) error {
	// 读取文件内容
	file, err := os.Open(file_path)
	if err != nil {
		return err
	}
	defer file.Close()
	// 创建一个临时文件来存储替换后的内容
	tmp_file_path := file_path + ".tmp"
	os.Remove(tmp_file_path)
	tmp_file, err := os.Create(tmp_file_path)
	if err != nil {
		return err
	}
	defer tmp_file.Close()

	replacements := map[string]string{
		`(?m)^172\.16\.\d+\.\d+`:  ip,
		`(?m)^192\.168\.\d+\.\d+`: ip,
		`(?m)^10\.\d+\.\d+\.\d+`:  ip,
		`(?m)^magician_ui_tag.*`:   fmt.Sprintf("magician_ui_tag = %s", ui_tag),
		`(?m)^magician_tag.*`:      fmt.Sprintf("magician_tag = %s", magician_tag),
	}

	// 用于分隔行的正则表达式
	re := regexp.MustCompile(`\r?\n`)
	buffer := make([]byte, 1024)
	var replace_content string
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		content := string(buffer[:n])
		for pattern, replace_str := range replacements {
			content = re.ReplaceAllString(content, "\n")
			content = regexp.MustCompile(pattern).ReplaceAllString(content, replace_str)
		}
		replace_content += content
	}

	//  将替换后的内容写入临时文件
	_, err = tmp_file.WriteString(replace_content)
	if err != nil {
		return err
	}

	// 移除原始文件，并将临时文件重命名为原始文件名
	err = os.Remove(file_path)
	if err != nil {
		return err
	}
	err = os.Rename(tmp_file_path, file_path)
	if err != nil {
		return err
	}
	return nil

}

func mody_sshd(remote_host, remote_username, private_key_path string) error {
	keyBytes, err := ioutil.ReadFile(private_key_path)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}
	// 配置ssh客户端
	config := &ssh.ClientConfig{
		User: remote_username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", remote_host+":22", config)
	if err != nil {
		return err
	}
	defer client.Close()
	// session, err := client.NewSession()
	// if err != nil {
	// 	return err
	// }
	// defer session.Close()
	// 执行远程命令
	commands := []string{
		"sudo sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/' /etc/ssh/sshd_config",
		"sudo sed -i 's/#UseDNS yes/UseDNS no/' /etc/ssh/sshd_config",
		"sudo systemctl restart sshd",
	}
	for _, cmd := range commands {
		session, err := client.NewSession()
		if err != nil {
			return err
		}
		if err := session.Run(cmd); err != nil {
			// if err := run_command(session, cmd); err != nil {
			return err
		}
		session.Close()
	}
	return nil
}

func push_directory(local_directory, remote_host, remote_username, private_key_path, remote_directory string) error {
	keyBytes, err := ioutil.ReadFile(private_key_path)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}
	// 配置ssh客户端
	config := &ssh.ClientConfig{
		User: remote_username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", remote_host+":22", config)
	if err != nil {
		return err
	}
	defer client.Close()
	//  建立sftp连接
	sftp_client, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftp_client.Close()
	//  递归上传目录
	err = filepath.Walk(local_directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// sftp_client.MkdirAll(remote_directory)
		// if info.IsDir() {
		// 	err = sftp_client.MkdirAll(remote_directory + "/" + info.Name())
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		rel_path, err := filepath.Rel(local_directory, path)
		if err != nil {
			return err
		}
		remote_path := filepath.Join(remote_directory, filepath.ToSlash(rel_path))
		// remote_path := filepath.Join(remote_directory, path)
		if info.IsDir() {
			// err = sftp_client.MkdirAll(remote_directory + "/" + info.Name())
			err = sftp_client.MkdirAll(remote_path)
			if err != nil {
				return err
			}
		} else {
			//  确保远程目录存在
			remote_dir := filepath.Dir(remote_path)
			err := sftp_client.MkdirAll(remote_dir)
			if err != nil {
				return err
			}
			local_file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer local_file.Close()
			remote_file, err := sftp_client.Create(remote_path)
			if err != nil {
				return err
			}
			defer remote_file.Close()
			fmt.Println("上传文件:", remote_path)
			_, err = io.Copy(remote_file, local_file)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func install_rpms(rpm_path, remote_host, remote_username, private_key_path string) error {
	keyBytes, err := ioutil.ReadFile(private_key_path)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}
	// 配置ssh客户端
	config := &ssh.ClientConfig{
		User: remote_username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", remote_host+":22", config)
	if err != nil {
		return err
	}
	defer client.Close()
	// session, err := client.NewSession()
	// if err != nil {
	// 	return err
	// }
	// defer session.Close()
	// 执行远程命令
	commands := []string{
		fmt.Sprintf("sudo yum install -y %s/roles/magician-deploy-4.9/files/tools/ansible/ansible_v2.9.9_install/*.rpm /root/magician-deployment/roles/magician-deploy-4.9/files/tools/ansible/ansible_v2.9.9_install/Ansible_install_RPM/*.rpm ", rpm_path),
		"sudo sed -i 's/#host_key_checking = False/host_key_checking = False/' /etc/ansible/ansible.cfg",
	}
	for _, cmd := range commands {
		session, err := client.NewSession()
		if err != nil {
			return err
		}
		if err := session.Run(cmd); err != nil {
			// if err := run_command(session, cmd); err != nil {
			return err
		}
		session.Close()
	}
	return nil
}

func install_magician(remote_directory, remote_host, remote_username, remote_password string) error {
	// keyBytes, err := ioutil.ReadFile(private_key_path)
	// if err != nil {
	// 	return err
	// }
	// key, err := ssh.ParsePrivateKey(keyBytes)
	// if err != nil {
	// 	return err
	// }
	// 配置ssh客户端
	config := &ssh.ClientConfig{
		User: remote_username,
		Auth: []ssh.AuthMethod{
			ssh.Password(remote_password),
			// ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", remote_host+":22", config)
	if err != nil {
		return err
	}
	defer client.Close()
	// session, err := client.NewSession()
	// if err != nil {
	// 	return err
	// }
	// defer session.Close()
	// 执行远程命令
	commands := []string{
		fmt.Sprintf("ansible-playbook -i  %s/hosts-magician-deploy-4.9 %s/magician-deploy-4.9.yml ", remote_directory, remote_directory),
	}
	for _, cmd := range commands {
		session, err := client.NewSession()
		if err != nil {
			return err
		}
		// 使用 session.CombinedOutput
		output, err := session.CombinedOutput(cmd)
		if err != nil {
			fmt.Println("命令执行失败:", err)

		} else {
			fmt.Println("命令输出:", string(output))
		}

		// if err := session.Run(cmd); err != nil {
		// 	// if err := run_command(session, cmd); err != nil {
		// 	return err
		// }
		session.Close()
	}
	return nil
}

func main() {
	// 获取脚本当前目录
	current_dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取脚本当前目录失败")
		return
	}
	fmt.Println("当前目录:", current_dir)

	// 构建配置文件的完整路径
	config_file_name := "config.yaml"
	config_file_path := filepath.Join(current_dir, config_file_name)
	v := viper.New()
	v.AddConfigPath(current_dir)
	v.SetConfigFile(config_file_path)
	v.SetConfigType("yaml")
	// 尝试读取配置文件
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("配置文件读取错误:", err)
		return
	}
	cpum := v.GetStringMapString("databases.zstack")["cpum"]
	image := v.GetStringMapString("databases.zstack")["image"]
	l3 := v.GetStringMapString("databases.zstack")["l3"]
	vm_name := v.GetStringMapString("databases.zstack")["vm_name"]

	// 密码
	password := sha512_hash("password")
	session_uuid, err := login(password)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("session_uuid:", session_uuid)
	// query_vm(session_uuid)
	// fmt.Println(sha512_hash("password"))
	cpum_uuid, err := query_instance_offering(session_uuid, cpum)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("cpum_uuid:", cpum_uuid)
	image_uuid, _ := query_image(session_uuid, image)
	fmt.Println("image_uuid:", image_uuid)
	l3_uuid, _ := query_l3(session_uuid, l3)
	fmt.Println("l3_uuid:", l3_uuid)
	// ip := create_vm(session_uuid, vm_name, cpum_uuid, image_uuid, l3_uuid)
	ip := "172.16.44.191"
	fmt.Println("ip:", ip, vm_name)
	service_names := v.GetStringMapString("databases.magician")["server"]
	names := strings.Split(service_names, ",")
	// for _, name := range strings.Splist(v.GetStringMapString("databases.magician")["server"], ",") {
	for _, name := range names {
		magician_tag := get_harbor_tag(v.GetStringMapString("databases.harbor")["ip"],
			name,
			v.GetStringMapString("databases.ansible")["magician_tag"])
		if !magician_tag {
			build_and_push_docker(v.GetStringMapString("databases.harbor")["username"],
				v.GetStringMapString("databases.harbor")["password"],
				name,
				v.GetStringMapString("databases.ansible")["magician_tag"],
				v.GetStringMapString("databases.harbor")["dockerfile"],
				v.GetStringMapString("databases.harbor")["ip"],
				path.Join(v.GetStringMapString("databases.ansible")["local_directory"], "roles/magician-deploy-4.9/files/app/images"))
		}
	}
	service_ui_names := v.GetStringMapString("databases.magician")["server_ui"]
	names = strings.Split(service_ui_names, ",")
	for _, name := range names {
		magician_tag := get_harbor_tag(v.GetStringMapString("databases.harbor")["ip"],
			name,
			v.GetStringMapString("databases.ansible")["magician_ui_tag"])
		if !magician_tag {
			build_and_push_docker(v.GetStringMapString("databases.harbor")["username"],
				v.GetStringMapString("databases.harbor")["password"],
				name,
				v.GetStringMapString("databases.ansible")["magician_ui_tag"],
				v.GetStringMapString("databases.harbor")["ui_dockerfile"],
				v.GetStringMapString("databases.harbor")["ip"],
				path.Join(v.GetStringMapString("databases.ansible")["local_directory"], "roles/magician-deploy-4.9/files/server/images/x86_64"))
		}
	}
	err = replace_ips(v.GetStringMapString("databases.ansible")["hostfile"], ip, v.GetStringMapString("databases.ansible")["magician_tag"], v.GetStringMapString("databases.ansible")["magician_ui_tag"])
	if err != nil {
		fmt.Println(err)
	}
	err = mody_sshd(ip, v.GetStringMapString("databases.ansible")["remote_username"], v.GetStringMapString("databases.ansible")["private_key_path"])
	if err != nil {
		fmt.Println(err)
	}
	push_directory(v.GetStringMapString("databases.ansible")["local_directory"], ip, v.GetStringMapString("databases.ansible")["remote_username"], v.GetStringMapString("databases.ansible")["private_key_path"], v.GetStringMapString("databases.ansible")["remote_directory"])
	install_rpms(v.GetStringMapString("databases.ansible")["remote_directory"], ip, v.GetStringMapString("databases.ansible")["remote_username"], v.GetStringMapString("databases.ansible")["private_key_path"])
	install_magician(v.GetStringMapString("databases.ansible")["remote_directory"], ip, v.GetStringMapString("databases.ansible")["remote_username"], v.GetStringMapString("databases.ansible")["remote_password"])
}

/*
func print_flag(cpum string, image string, l3 string) {
	fmt.Println(cpum)
	fmt.Println(image)
	fmt.Println(l3)
}
*/
