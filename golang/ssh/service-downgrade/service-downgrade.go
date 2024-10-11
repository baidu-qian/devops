/*
 * @Author: magician
 * @Date: 2024-09-20 00:57:05
 * @LastEditors: magician
 * @LastEditTime: 2024-10-10 23:56:01
 * @FilePath: /go/src/go_code/ssh/service-downgrade/service-downgrade.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

func get_remote_system_info(ip, username, password string, port int) (string, string, string, error) {
	/**
	@brief 获取远程机器系统信息
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@return string cpuinfo string
	@return string meminfo string
	@return string disk_output string
	@return error error
	**/

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程机器
	hosts := ip + ":" + fmt.Sprintf("%d", port)
	client, err := ssh.Dial("tcp", hosts, config)
	if err != nil {
		fmt.Println("Failed to dial: ", err)
		return "", "", "", err
	}
	defer client.Close()

	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return "", "", "", err
	}
	defer session.Close()
	// 执行远程命令
	output, err := session.CombinedOutput("top -b -n 1 | grep 'Cpu(s)'")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	cpuinfo := strings.Split(string(output), ",")[3]
	cpuinfo = strings.Split(cpuinfo, "id")[0]

	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return "", "", "", err
	}
	defer session.Close()
	// 执行远程命令
	output, err = session.CombinedOutput("free -g | grep Mem")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	meminfo := strings.Fields(string(output))[5]

	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return "", "", "", err
	}
	defer session.Close()
	// 执行远程命令
	output, err = session.CombinedOutput("df -h -BG /home/app | sort -k 4 -h|tail -n 1")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	disk_output := strings.Fields(string(output))[3]
	disk_output = strings.TrimSuffix(disk_output, "G")
	return cpuinfo, meminfo, disk_output, nil
}

func get_es_client_mem(ip, username, password string, port int) (int, int, error) {
	/**
	@brief 获取es client内存
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@return int 内存大小
	@return error error
	**/

	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		fmt.Println("无法连接到 %s: %s", addr, err)
		return 0, 0, err
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return 0, 0, err
	}
	defer session.Close()
	// 使用 SFTP 传输文件
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		log.Fatalf("Failed to create SFTP client: %s", err)
	}
	defer sftpClient.Close()

	// 打开文件并读取内容
	filePath := "/home/app/server/elasticsearchClient/elasticsearch/config/jvm.options"
	file, err := sftpClient.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}

	lines := strings.Split(string(content), "\n")

	var xms, xmx int
	for _, line := range lines {
		if strings.HasPrefix(line, "-Xms") {
			xms_str := strings.TrimSpace(line[4 : len(line)-1])
			xms, _ = strconv.Atoi(xms_str)
		} else if strings.HasPrefix(line, "-Xmx") {
			xmx_str := strings.TrimSpace(line[4 : len(line)-1])
			xmx, _ = strconv.Atoi(xmx_str)
		}
	}

	return xms, xmx, nil
}

func set_es_client_mem(ip, username, password string, port, xms, xmx int) error {
	/**
	@brief 设置es client内存
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@param xms int 内存大小
	@param xmx int 内存大小
	@return error error
	**/

	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
	address := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
	}
	defer client.Close()

	// 使用 SFTP 传输文件
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %s", err)
	}
	defer sftpClient.Close()

	// 打开文件并读取内容
	filePath := "/home/app/server/elasticsearchClient/elasticsearch/config/jvm.options"
	file, err := sftpClient.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %s", err)
	}

	lines := strings.Split(string(content), "\n")

	// 修改内存值
	for i, line := range lines {
		if strings.HasPrefix(line, "-Xms") {
			lines[i] = fmt.Sprintf("-Xms%dg", xms) // 修改为新的值，使用 G 表示千兆字节
		} else if strings.HasPrefix(line, "-Xmx") {
			lines[i] = fmt.Sprintf("-Xmx%dg", xmx) // 修改为新的值，使用 G 表示千兆字节
		}
	}

	// 将修改后的内容写回文件
	newContent := strings.Join(lines, "\n")
	tmempFilePath := filepath.Join(os.TempDir(), "jvm.options.tmp")
	err = ioutil.WriteFile(tmempFilePath, []byte(newContent), 0644) // 使用0644权限写入文件
	if err != nil {
		return fmt.Errorf("写入临时文件失败: %s", err)
	}
	// 打开临时文件并上传到远程服务器
	localFile, err := os.Open(tmempFilePath)
	if err != nil {
		return fmt.Errorf("打开临时文件失败: %s", err)
	}
	defer localFile.Close()

	err = sftpClient.Remove(filePath)
	if err != nil {
		return fmt.Errorf("重命名文件失败: %s", err)
	}
	remoteFile, err := sftpClient.Create(filePath) // 创建远程文件
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %s", err)
	}
	defer remoteFile.Close()
	// 将本地内容写入到远程文件
	if _, err = io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("上传到远程文件失败: %s", err)
	}

	return nil
}

func server_stop(ip, username, password string, port int, server_name string) error {
	/**
	@brief 停止服务
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@return error error
	**/
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return err
	}
	if server_name == "elasticsearch" {
		_, err = session.CombinedOutput("bash /home/app/server/elasticsearchClient/bin/service.sh stop")
		if err != nil {
			return err
		}
	}
	return nil
}

func server_restart(ip, username, password string, port int, app_name string) error {
	/*
		@brief 应用重启
		@param ip string 服务器ip
		@param username string 用户名
		@param password string 密码
		@param port int 端口号
		@param app_name string 应用名称
	**/

	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
	}
	defer session.Close()
	_, err = session.CombinedOutput("bash /home/app/server/" + app_name + "/bin/service.sh stop")
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return err
	}
	defer session.Close()
	_, err = session.CombinedOutput("bash /home/app/server/" + app_name + "/bin/service.sh start")
	if err != nil {
		return err
	}
	return nil
}

func app_stop(ip, username, password string, port int, app_name string) error {
	/**
	@brief 应用停止
	@param ip string 服务器ip
	@param username string 用户名
	@param password string 密码
	@param port int 端口号
	@param app_name string 应用名称
	**/
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return err
	}
	defer session.Close()

	_, err = session.CombinedOutput("bash /home/app/app/" + app_name + "/bin/service.sh stop")
	if err != nil {
		return err
	}
	return nil
}

func check_remote_file_exists(ip, username, password string, port int, filePath string) (bool, error) {
	/**
	@brief 检查远程文件是否存在
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@param filePath string 文件路径
	@return bool 是否存在
	@return error error
	**/
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
		return false, err
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return false, err
	}
	defer session.Close()
	_, err = session.CombinedOutput("test -e " + filePath)
	if err != nil {
		return false, err
	}
	return true, nil
}

func find_files(ip, username, password string, port int, filePath string) (bool, error) {
	/**
	@brief 查找远程是否有hs_err文件
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@param filePath string 文件路径
	@return bool 是否存在
	@return error error
	**/
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
		return false, err
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return false, err
	}
	defer session.Close()
	output, err := session.CombinedOutput("find " + filePath + " -name *.yaml")
	if err != nil {
		return false, err
	}
	if len(output) > 0 {
		return true, nil
	}
	return false, nil
}

func rm_files(ip, username, password string, port int, filePath string) (bool, error) {
	/**
	@brief 删除远程文件
	@param ip string 远程机器IP
	@param username string 远程机器用户名
	@param password string 远程机器密码
	@param port int 远程机器端口
	@param filePath string 文件路径
	@return error error
	**/
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal("无法连接到 %s: %s", addr, err)
		return false, err
	}
	defer client.Close()
	// 执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return false, err
	}
	defer session.Close()
	_, err = session.CombinedOutput("find " + filePath + " -name hs_err* | xargs -i rm -f {}")
	if err != nil {
		return false, err
	}
	return true, nil
}

func main() {
	current_dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取脚本当前目录失败")
		return
	}
	config_file_name := "config/config.yaml"
	config_file_path := filepath.Join(current_dir, config_file_name)
	v := viper.New()
	v.AddConfigPath(current_dir)
	v.SetConfigFile(config_file_path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("配置文件读取错误:", err)
		return
	}

	es_ip := v.Get("database.es").([]interface{})
	username := v.GetStringMapString("database.ssh")["username"]
	password := v.GetStringMapString("database.ssh")["password"]
	port_str := v.GetStringMapString("database.ssh")["port"]
	port, _ := strconv.Atoi(port_str)
	for {
		for _, list := range es_ip {
			config := list.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				cpuinfo_str, meminfo_str, disk_output_str, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_int, _ := strconv.Atoi(cpuinfo_str)
				meminfo_int, _ := strconv.Atoi(meminfo_str)
				fmt.Printf("cpuinfo: %s, meminfo: %s G, disk_output: %s G\n", cpuinfo_str, meminfo_str, disk_output_str)
				file_exists, _ := find_files(ip, username, password, port, "/home/app")
				if (cpuinfo_int < 20 && meminfo_int < 2) || file_exists {
					fmt.Println("cpu或者内存不足,需要降级")
					// 降级es
					es_xms, es_xmx, _ := get_es_client_mem(ip, username, password, port)
					if es_xms > 4 {
						set_es_client_mem(ip, username, password, port, es_xms/2, es_xmx/2)
						server_restart(ip, username, password, port, "elasticSearchClient")
					} else {
						set_es_client_mem(ip, username, password, port, 2, 2)
						server_restart(ip, username, password, port, "elasticSearchClient")
					}
					status, err := rm_files(ip, username, password, port, "/home/app")
					if !status || err != nil {
						fmt.Println("删除文件失败")
					}
				}

			}
		}
		// cpuinfo_str, meminfo_str, disk_output_str, _ := get_remote_system_info(es_ip, username, password, port)
		// fmt.Printf("cpuinfo: %s, meminfo: %s G, disk_output: %s G\n", cpuinfo_str, meminfo_str, disk_output_str)
		for _, list := range v.Get("database.app-sender").([]interface{}) {
			config := list.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				cpuinfo_str, meminfo_str, _, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_int, _ := strconv.Atoi(cpuinfo_str)
				meminfo_int, _ := strconv.Atoi(meminfo_str)
				file_exists, _ := find_files(ip, username, password, port, "/home/app")
				if (cpuinfo_int < 20 && meminfo_int < 2) || file_exists {
					fmt.Println("cpu或者内存不足,需要降级")
					status, _ := check_remote_file_exists(ip, username, password, port, "/home/app/app/app-sender")
					if status {
						app_stop(ip, username, password, port, "app-sender")
					}
					status, err := rm_files(ip, username, password, port, "/home/app")
					if !status || err != nil {
						fmt.Println("删除文件失败")
					}
				}
			}
		}

		for _, list := range v.Get("database.security-event").([]interface{}) {
			config := list.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				cpuinfo_str, meminfo_str, _, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_int, _ := strconv.Atoi(cpuinfo_str)
				meminfo_int, _ := strconv.Atoi(meminfo_str)
				file_exists, _ := find_files(ip, username, password, port, "/home/app")
				if (cpuinfo_int < 20 && meminfo_int < 2) || file_exists {
					fmt.Println("cpu或者内存不足,需要降级")
					status, _ := check_remote_file_exists(ip, username, password, port, "/home/app/app/security-event")
					if status {
						app_stop(ip, username, password, port, "security-event")
					}
					status, err := rm_files(ip, username, password, port, "/home/app")
					if !status || err != nil {
						fmt.Println("删除文件失败")
					}
				}
			}
		}

		for _, list := range v.Get("database.threat").([]interface{}) {
			config := list.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				cpuinfo_str, meminfo_str, _, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_int, _ := strconv.Atoi(cpuinfo_str)
				meminfo_int, _ := strconv.Atoi(meminfo_str)
				file_exists, _ := find_files(ip, username, password, port, "/home/app")
				if (cpuinfo_int < 20 && meminfo_int < 2) || file_exists {
					fmt.Println("cpu或者内存不足,需要降级")
					status, _ := check_remote_file_exists(ip, username, password, port, "/home/app/app/threat")
					if status {
						app_stop(ip, username, password, port, "threat")
					}
					status, err := rm_files(ip, username, password, port, "/home/app")
					if !status || err != nil {
						fmt.Println("删除文件失败")
					}
				}
			}
		}

		for _, list := range v.Get("database.analyzer-dev").([]interface{}) {
			config := list.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				cpuinfo_str, meminfo_str, _, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_int, _ := strconv.Atoi(cpuinfo_str)
				meminfo_int, _ := strconv.Atoi(meminfo_str)
				file_exists, _ := find_files(ip, username, password, port, "/home/app")
				if (cpuinfo_int < 20 && meminfo_int < 2) || file_exists {
					fmt.Println("cpu或者内存不足,需要降级")
					status, _ := check_remote_file_exists(ip, username, password, port, "/home/app/app/analyzer-dev")
					if status {
						app_stop(ip, username, password, port, "analyzer-dev")
					}
					status, err := rm_files(ip, username, password, port, "/home/app")
					if !status || err != nil {
						fmt.Println("删除文件失败")
					}
				}
			}
		}
		time.Sleep(5 * time.Minute)
	}
}
