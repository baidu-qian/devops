/*
 * @Author: magician
 * @Date: 2024-08-27 23:33:24
 * @LastEditors: magician
 * @LastEditTime: 2024-09-11 00:28:43
 * @FilePath: /go/src/go_code/ssh/auto-replicas/auto-replicas.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

func get_kafka_lag(brokerList []string, limit int64) (map[string]bool, error) {
	/**
	@brief 获取kafka groupid
	@param brokerList []string kafka服务器地址
	@param limit int64 lag大小
	@return map[string]bool groupid
	@return error
	**/
	config := sarama.NewConfig()
	config.Version = sarama.V2_1_0_0
	config.Consumer.Return.Errors = true

	client, err := sarama.NewClient(brokerList, config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return nil, err
	}
	defer admin.Close()

	// 读取kafka所有的groupid
	// groupIds, err := admin.ListConsumerGroups()
	// if err != nil {
	// 	return nil, err
	// }

	// for groupId, _ := range groupIds {
	// 	fmt.Println(groupId)
	// }

	type Result struct {
		groupId string
		lag     map[string]int64
		err     error
	}

	wxgz_groups := []string{"groupid_threat", "groupid_dataservice", "event_data_preparation", "cleaners_groupid", "groupid_running_distribution"}
	// lags := make(map[string]map[string]int64)
	results := make(chan Result, len(wxgz_groups))
	var wg sync.WaitGroup

	for _, groupId := range wxgz_groups {
		wg.Add(1)
		go func(groupId string) {
			defer wg.Done()
			groupLag := make(map[string]int64)
			consecutiveLags := 0
			for i := 0; i < 3; i++ {

				groups, err := admin.ListConsumerGroupOffsets(groupId, nil)
				if err != nil {
					results <- Result{groupId, nil, err}
					return
				}
				totalLag := int64(0)
				for topic, partitions := range groups.Blocks {
					for partition, block := range partitions {
						newestOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
						if err != nil {
							results <- Result{groupId, nil, err}
							return
						}
						groupLag[topic] += newestOffset - block.Offset
					}
				}
				if totalLag >= limit {
					consecutiveLags++
					time.Sleep(5 * time.Minute)
					// time.Sleep(5 * time.Second)
				} else {
					consecutiveLags = 0
				}
			}
			if consecutiveLags >= 3 {
				results <- Result{groupId, groupLag, nil}
			}
		}(groupId)
	}

	wg.Wait()
	close(results)

	// 处理结果
	uniqueGroupIds := make(map[string]bool)
	for result := range results {
		if result.err != nil {
			fmt.Printf("Error processing group %s: %v\n", result.groupId, result.err)
		} else {
			uniqueGroupIds[result.groupId] = true
		}
	}

	return uniqueGroupIds, nil
}

func get_remote_system_info(ip, username, password string, port int) (string, string, string, error) {
	/**
	@brief 获取远程系统信息
	@param ip string 服务器ip
	@param username string 用户名
	@param password string 密码
	@param port int 端口号
	@return string CPU信息
	@return string 内存信息
	@return string 磁盘信息
	**/

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
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
	// fmt.Printf("cpu可用为: %s\n", cpuinfo)

	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return "", "", "", err
	}
	defer session.Close()
	output, err = session.CombinedOutput("free -g|grep Mem")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	meminfo := strings.Fields(string(output))[6]
	// fmt.Printf("mem可用为: %s\n", meminfo)

	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return "", "", "", err
	}
	defer session.Close()
	output, err = session.CombinedOutput("df -h -BG /home/app | sort -k 4 -h|tail -n 1")
	if err != nil {
		fmt.Println("Failed to run: ", err)
		return "", "", "", err
	}
	disk_output := strings.Fields(string(output))[3]
	disk_output = strings.TrimSuffix(disk_output, "G")
	// fmt.Printf("硬盘可用为: %s\n", disk_output)
	return cpuinfo, meminfo, disk_output, nil

}

func get_app_replicas(ip, username, password string, port int, app_name string) (int, error) {
	/*
		@brief 获取应用副本数
		@param ip string 服务器ip
		@param username string 用户名
		@param password string 密码
		@param port int 端口号
		@param app_name string 应用名称
		@return int 应用副本数
		**/
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {

		log.Fatal("Failed to dial: ", err)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal("Failed to create client: ", err)
	}
	defer client.Close()
	remotePath := filepath.Join("/home/app/app", app_name, "bin/docker-compose.yml")
	file, err := client.Open(remotePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	// Unmarshal YAML data
	var composeData map[string]interface{}
	if err := yaml.Unmarshal(content, &composeData); err != nil {
		log.Fatalf("Failed to unmarshal YAML: %s", err)
	}
	replicas := int(0)
	// Update the 'replicas' field for the specified service
	if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
		for _, service := range services {
			if deploy, ok := service.(map[interface{}]interface{})["deploy"].(map[interface{}]interface{}); ok {
				replicas = deploy["replicas"].(int)

				// fmt.Println(deploy["replicas"])
				break
			}
		}
	}
	return replicas, nil
}

func set_app_replicas(ip, username, password string, port int, app_name string, replicas int) {
	/*
		@brief 修改应用副本数
		@param ip string 服务器ip
		@param username string 用户名
		@param password string 密码
		@param port int 端口号
		@param app_name string 应用名称
		@param replicas int 副本数
		**/
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {

		log.Fatal("Failed to dial: ", err)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal("Failed to create client: ", err)
	}
	defer client.Close()
	remotePath := filepath.Join("/home/app/app", app_name, "bin/docker-compose.yml")
	file, err := client.Open(remotePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	// Unmarshal YAML data
	var composeData map[string]interface{}
	if err := yaml.Unmarshal(content, &composeData); err != nil {
		log.Fatalf("Failed to unmarshal YAML: %s", err)
	}
	// Update the 'replicas' field for the specified service
	if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
		for _, service := range services {
			if deploy, ok := service.(map[interface{}]interface{})["deploy"].(map[interface{}]interface{}); ok {
				deploy["replicas"] = replicas
				// fmt.Println(deploy["replicas"])
				break
			}
		}
	}
	updateData, err := yaml.Marshal(composeData)
	if err != nil {
		log.Fatal("Failed to marshal YAML: ", err)
	}
	file, err = client.Create(remotePath)
	if err != nil {
		log.Fatal("Failed to write file: ", err)
	}
	defer file.Close()

	if _, err := file.Write(updateData); err != nil {
		log.Fatal("Failed to write file: ", err)
	}
}

func app_restart(ip, username, password string, port int, app_name string) {
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
		Timeout:         30 * time.Second,
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
		return
	}
	defer session.Close()
	_, err = session.CombinedOutput("/home/app/bin/docker-compose -f /home/app/app/" + app_name + "/bin/docker-compose.yml down")
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Print(output)
	time.Sleep(2 * time.Second)
	// 执行远程命令
	session, err = client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return
	}
	defer session.Close()
	_, err = session.CombinedOutput("/home/app/bin/docker-compose -f /home/app/app/" + app_name + "/bin/docker-compose.yml up -d")
	if err != nil {
		log.Printf("启动服务失败: %v", err)
	}
	// fmt.Print(output)
}

func deploy_app_replicas(v *viper.Viper, app_name string) {
	/**
		@brief 部署应用副本数
		@param app_name string 应用名称
	**/
	username := v.GetStringMapString("database.ssh")["username"]
	password := v.GetStringMapString("database.ssh")["password"]
	port_str := v.GetStringMapString("database.ssh")["port"]
	port, _ := strconv.Atoi(port_str)
	replicas := v.Get("database.replicas").(map[string]interface{})
	limit_cpu_str := v.GetStringMapString("database.limit")["cpu"]
	limit_cpu, _ := strconv.ParseFloat(limit_cpu_str, 32)
	limit_mem_str := v.GetStringMapString("database.limit")["memory"]
	limit_mem, _ := strconv.Atoi(limit_mem_str)
	limit_disk_str := v.GetStringMapString("database.limit")["disk"]
	limit_disk, _ := strconv.ParseFloat(limit_disk_str, 32)

	// 检查 appName 是否存在于 replicas 中
	if service, exists := replicas[app_name]; exists {
		// 将 service 转换为切片
		for _, serviceConfig := range service.([]interface{}) {
			config := serviceConfig.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				// min := config["min"].(int)
				max := config["max"].(int)
				cpuinfo_str, meminfo_str, disk_output_str, _ := get_remote_system_info(ip, username, password, port)
				cpuinfo_str = strings.TrimSpace(cpuinfo_str)
				cpuinfo, _ := strconv.ParseFloat(cpuinfo_str, 32)
				meminfo, _ := strconv.Atoi(meminfo_str)
				disk_output, _ := strconv.ParseFloat(disk_output_str, 32)
				if cpuinfo >= limit_cpu && meminfo >= limit_mem && disk_output >= limit_disk {
					fmt.Printf("cpuinfo: %v\nmeminfo: %v\ndisk_output: %v\n", cpuinfo, meminfo, disk_output)
					replicas, _ := get_app_replicas(ip, username, password, port, app_name)
					fmt.Printf("[%s]扩容前: 当前副本数:%d\n", app_name, replicas)
					if replicas < max {
						set_app_replicas(ip, username, password, port, app_name, replicas+1)
						app_restart(ip, username, password, port, app_name)
						replicas, _ = get_app_replicas(ip, username, password, port, app_name)
						fmt.Printf("[%s]扩容后: 当前副本数:%d\n", app_name, replicas)
					}
				}
			}
		}
	}

}

func fallback_app_replicas(v *viper.Viper, app_name string) {
	/**
		@brief 回退应用副本数
		@param app string 应用名称
	**/
	username := v.GetStringMapString("database.ssh")["username"]
	password := v.GetStringMapString("database.ssh")["password"]
	port_str := v.GetStringMapString("database.ssh")["port"]
	port, _ := strconv.Atoi(port_str)
	replicas := v.Get("database.replicas").(map[string]interface{})
	// 检查 appName 是否存在于 replicas 中
	if service, exists := replicas[app_name]; exists {
		// 将 service 转换为切片
		for _, serviceConfig := range service.([]interface{}) {
			// if serviceConfig.(map[string]interface{}).["enable"] {
			config := serviceConfig.(map[string]interface{})
			if config["enable"].(bool) {
				ip := config["ip"].(string)
				min := config["min"].(int)
				// max := config["max"]
				replicas, _ := get_app_replicas(ip, username, password, port, app_name)
				fmt.Printf("[%s]缩容前: 当前副本数:%d\n", app_name, replicas)
				if replicas > min {
					set_app_replicas(ip, username, password, port, app_name, replicas-1)
					app_restart(ip, username, password, port, app_name)
					replicas, _ = get_app_replicas(ip, username, password, port, app_name)
					fmt.Printf("[%s]扩容后: 当前副本数:%d\n", app_name, replicas)
				}
			}
		}
	}
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
	kafka_brokers := v.GetStringMapString("database.kafka")["brokers"]
	lag_str := v.GetStringMapString("database.kafka")["lag"]
	lag, _ := strconv.ParseInt(lag_str, 10, 64)

	for {
		groups, _ := get_kafka_lag(strings.Split(kafka_brokers, ","), lag)
		for groupId := range groups {
			switch groupId {
			case "cleaners_groupid":
				deploy_app_replicas(v, "cleaner")
			case "groupid_dataservice":
				deploy_app_replicas(v, "transfer")
			case "event_data_preparation":
				deploy_app_replicas(v, "security-event")
			case "groupid_threat":
				deploy_app_replicas(v, "threat")
			case "groupid_running_distribution":
				deploy_app_replicas(v, "analyzer-dev")
			default:
				fmt.Println("未知主题, 无需扩容")
			}
		}
		wxgz_groups := []string{"cleaner", "transfer", "security-event", "threat", "analyzer-dev"}
		if len(groups) < len(wxgz_groups) {
			fallback_app_tag := []string{}
			for groupId := range groups {
				switch groupId {
				case "cleaners_groupid":
					fallback_app_tag = append(fallback_app_tag, "cleaner")
				case "groupid_dataservice":
					fallback_app_tag = append(fallback_app_tag, "transfer")
				case "event_data_preparation":
					fallback_app_tag = append(fallback_app_tag, "security-event")
				case "groupid_threat":
					fallback_app_tag = append(fallback_app_tag, "threat")
				case "groupid_running_distribution":
					fallback_app_tag = append(fallback_app_tag, "analyzer-dev")
				default:
					fmt.Println("未知主题, 无需缩容")
				}
			}
			for _, app := range wxgz_groups {
				// 如果_app不存在于fallback_app_tags中，执行缩容操作
				if !slices.Contains(fallback_app_tag, app) {
					fallback_app_replicas(v, app)
					time.Sleep(10 * time.Second)
				}
			}
		}
		time.Sleep(5 * time.Minute)
	}
}
