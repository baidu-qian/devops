/*
 * @Author: magician
 * @Date: 2024-09-20 00:57:05
 * @LastEditors: magician
 * @LastEditTime: 2024-09-22 11:06:20
 * @FilePath: /go/src/go_code/ssh/service-downgrade/service-downgrade.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package make

import (
	"fmt"
	"strings"
	"time"

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
	output, err = session.CombinedOutput("free -m | grep Mem")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	meminfo := strings.Fields(string(output))[5]

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
