/*
 * @Author: magician
 * @Date: 2025-02-09 21:22:15
 * @LastEditors: magician
 * @LastEditTime: 2025-03-05 00:57:55
 * @FilePath: /go_code/DisasterRecovery/everisk.go
 * @Description:
 *
 * Copyright (c) 2025 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

var logger *log.Logger

// 定义TABLE_FAMILY_MAP
var TABLE_FAMILY_MAP = map[string]string{
	"admin_cleaner_start_cache":                     "ttl_family",
	"admin_config_sync":                             "info",
	"admin_dev_location_status":                     "status",
	"admin_dev_status":                              "dev_status",
	"admin_dev_threat_index_data":                   "data",
	"admin_fdti_data":                               "fdti",
	"admin_fdti_hxb_level":                          "fdti",
	"admin_last_start_message":                      "data",
	"admin_push_data_hxb_field_pool":                "common_data,threat_env",
	"admin_threat_illegal_app_statistical_learning": "statistical_learning",
	"admin_threat_index":                            "data",
	"admin_threat_index_data":                       "data",
	"admin_threat_index_data_sync_status":           "data",
	"admin_weekly_active_device":                    "status",
	"bb_apkinfo_cache":                                "content",
	"bb_ccb_push_service_cache":                       "security_event",
	"bb_dev_app_information_history":                  "information",
	"bb_dev_info_cache":                               "content",
	"bb_dev_info_current":                             "information",
	"bb_modem_pool_cache":                             "content",
	"bb_new_dev_fingerprint_factor_cache":             "associate_id",
	"bb_security_event_data_track":                    "track",
	"bb_security_event_fact_id_track":                 "counter,fact_id",
	"bb_security_event_fact_risk":                     "",
	"bb_security_event_origin_record":                 "",
	"bb_start_commons_fields":                         "common_fields",
}

// update_json 编辑 JSON 数据
//
// Args:
//
//	jsonData: JSON 数据
//	v: 配置文件
//
// Returns:
//
//	编辑后的 JSON 数据
func update_json(jsonData map[string]interface{}, v *viper.Viper) map[string]interface{} {
	// 更新数据库
	if v.GetBool("database.db.enable") {
		for key, value := range v.Get("database.db").(map[string]interface{}) {
			if key != "enable" {
				if used, ok := jsonData["global"].(map[string]interface{})["database"].(map[string]interface{})["used"].(map[string]interface{}); ok {
					if _, exists := used[key]; exists {
						used[key] = value
					}
				}
			}
		}
	}

	// 添加 rsync
	if v.GetBool("database.transfer_rsync_es.enable") {
		for key, value := range v.Get("database.transfer_rsync_es").(map[string]interface{}) {
			if key != "enable" {
				if _, exists := jsonData["transfer_rsync_es"]; !exists {
					jsonData["transfer_rsync_es"] = make(map[string]interface{})
				}
				jsonData["transfer_rsync_es"].(map[string]interface{})[key] = value
			}
		}
	}

	// 更新 redis
	if v.GetBool("database.redis.enable") {
		if activeMode := v.GetString("database.redis.active_mode"); activeMode != "" {
			jsonData["global"].(map[string]interface{})["redis"].(map[string]interface{})["used"].(map[string]interface{})["active_mode"] = activeMode
			if activeMode == "single" {
				for key, value := range v.Get("database.redis.single").(map[string]interface{}) {
					// for key, value := range db["redis"].(map[string]interface{})["single"].(map[string]interface{}) {
					if _, exists := jsonData["global"].(map[string]interface{})["redis"].(map[string]interface{})["single"].(map[string]interface{})[key]; exists {
						jsonData["global"].(map[string]interface{})["redis"].(map[string]interface{})["single"].(map[string]interface{})[key] = value
					}
				}
			} else if activeMode == "cluster" {
				// for key, value := range db["redis"].(map[string]interface{})["cluster"].(map[string]interface{}) {
				for key, value := range v.Get("database.redis.cluster").(map[string]interface{}) {
					if _, exists := jsonData["global"].(map[string]interface{})["redis"].(map[string]interface{})["cluster"].(map[string]interface{})[key]; exists {
						jsonData["global"].(map[string]interface{})["redis"].(map[string]interface{})["cluster"].(map[string]interface{})[key] = value
					}
				}
			}
		}
	}

	// 更新 kibana
	// for key, value := range db["kibana"].(map[string]interface{}) {
	for key, value := range v.Get("database.kibana").(map[string]interface{}) {
		if _, exists := jsonData["global"].(map[string]interface{})["kibana"].(map[string]interface{})[key]; exists {
			jsonData["global"].(map[string]interface{})["kibana"].(map[string]interface{})[key] = value
		}
	}
	loginInfo, _ := json.Marshal(map[string]string{
		// "username": db["elasticsearch"].(map[string]interface{})["username"].(string),
		// "password": db["elasticsearch"].(map[string]interface{})["password"].(string),
		"username": v.GetString("database.elasticsearch.username"),
		"password": v.GetString("database.elasticsearch.password"),
	})
	jsonData["global"].(map[string]interface{})["kibana"].(map[string]interface{})["everisk.kibana.login"] = string(loginInfo)

	// 更新 elasticsearch
	// if esEnable, ok := db["elasticsearch"].(map[string]interface{})["enable"].(bool); ok && esEnable {
	if v.GetBool("database.elasticsearch.enable") {
		// for key, value := range db["elasticsearch"].(map[string]interface{}) {
		for key, value := range v.Get("database.elasticsearch").(map[string]interface{}) {
			if key != "enable" && key != "username" && key != "password" {
				if _, exists := jsonData["global"].(map[string]interface{})["elasticsearch"].(map[string]interface{})[key]; exists {
					jsonData["global"].(map[string]interface{})["elasticsearch"].(map[string]interface{})[key] = value
				}
			}
		}
		jsonData["global"].(map[string]interface{})["elasticsearch"].(map[string]interface{})["everisk.elasticsearch.login"] =
			// db["elasticsearch"].(map[string]interface{})["username"].(string) + ":" + db["elasticsearch"].(map[string]interface{})["password"].(string)
			v.GetString("database.elasticsearch.username") + ":" + v.GetString("database.elasticsearch.password")
	}

	return jsonData
}

// ssh_init_everisk_ha 通过 SSH 和 SFTP 连接远程服务器，读取并更新指定的 JSON 配置文件。
//
// 参数:
// - ip: 远程服务器的 IP 地址
// - username: 用于 SSH 认证的用户名
// - password: 用于 SSH 认证的密码
// - port: SSH 服务的端口号
// - v: Viper 配置实例，用于获取更新 JSON 所需的配置信息
//
// 功能:
// 连接到远程服务器，使用 SFTP 读取指定路径下的 config.json 文件，
// 解析并更新 JSON 数据，最后将更新后的 JSON 数据写回到远程服务器。

func ssh_init_everisk_ha(ip, username, password string, port int, v *viper.Viper) {
	// 创建ssh客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	// 连接远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接远程服务器: %v", err)
	}
	defer client.Close()
	// 创建sftp客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Fatalf("无法创建sftp客户端: %v", err)
	}
	defer sftpClient.Close()
	// 读取远程文件
	remoteFilePath := fmt.Sprintf("/home/%s/app/init/config/config.json", username)
	remoteFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		logger.Fatalf("无法打开远程文件: %v", err)
	}
	defer remoteFile.Close()

	// 解析 JSON 数据
	var jsonData map[string]interface{}
	if err := json.NewDecoder(remoteFile).Decode(&jsonData); err != nil {
		logger.Fatalf("无法解析 JSON 数据: %v", err)
	}

	// 更新 JSON 数据
	newJSONData := update_json(jsonData, v)

	// 将更新后的 JSON 数据写入临时文件
	tmpFilePath := fmt.Sprintf("/home/%s/app/init/config/config.json.tmp", username)
	tmpFile, err := sftpClient.Create(tmpFilePath)
	if err != nil {
		logger.Fatalf("无法创建临时文件: %v", err)
	}
	defer tmpFile.Close()

	// 将 JSON 数据写入临时文件
	jsonBytes, err := json.MarshalIndent(newJSONData, "", "  ")
	if err != nil {
		logger.Fatalf("无法序列化 JSON 数据: %v", err)
	}
	// if _, err := tmpFile.Write(jsonBytes); err != nil {
	// 	logger.Fatalf("无法写入临时文件: %v", err)
	// }
	// 在写入 JSON 文件时使用 pretty.Pretty 保持格式
	formattedJSON := pretty.Pretty(jsonBytes)
	if _, err := tmpFile.Write(formattedJSON); err != nil {
		logger.Fatalf("无法写入临时文件: %v", err)
	}

	// 先删除已存在文件，再重命名临时文件为正式文件
	if _, err := sftpClient.Stat(remoteFilePath); err == nil {
		if err := sftpClient.Remove(remoteFilePath); err != nil {
			logger.Fatalf("无法删除已存在文件: %v", err)
		}
	}
	if err := sftpClient.Rename(tmpFilePath, remoteFilePath); err != nil {
		logger.Fatalf("无法重命名文件: %v", err)
	}

	logger.Println("更新 config.json 成功")
}

func local_ssh_init_everisk_ha(username string, v *viper.Viper) {
	// 本地文件路径
	localFilePath := filepath.Join("/home", username, "app", "init", "config", "config.json")

	// 检查本地文件是否存在
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		logger.Fatalf("本地文件不存在: %v", localFilePath)
	}

	// 打开本地文件
	localFile, err := os.Open(localFilePath)
	if err != nil {
		logger.Fatalf("无法打开本地文件: %v", err)
	}
	defer localFile.Close()

	// 解析 JSON 数据
	var jsonData map[string]interface{}
	if err := json.NewDecoder(localFile).Decode(&jsonData); err != nil {
		logger.Fatalf("无法解析 JSON 数据: %v", err)
	}

	// 更新 JSON 数据
	newJSONData := update_json(jsonData, v)

	// 将更新后的 JSON 数据写入临时文件
	tmpFilePath := filepath.Join("/home", username, "app", "init", "config", "config.json.tmp")
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		logger.Fatalf("无法创建临时文件: %v", err)
	}
	defer tmpFile.Close()

	// 将 JSON 数据写入临时文件
	jsonBytes, err := json.MarshalIndent(newJSONData, "", "  ")
	if err != nil {
		logger.Fatalf("无法序列化 JSON 数据: %v", err)
	}
	formattedJSON := pretty.Pretty(jsonBytes)
	if _, err := tmpFile.Write(formattedJSON); err != nil {
		logger.Fatalf("无法写入临时文件: %v", err)
	}

	// 先删除已存在文件，再重命名临时文件为正式文件
	if _, err := os.Stat(localFilePath); err == nil {
		if err := os.Remove(localFilePath); err != nil {
			logger.Fatalf("无法删除已存在文件: %v", err)
		}
	}
	if err := os.Rename(tmpFilePath, localFilePath); err != nil {
		logger.Fatalf("无法重命名文件: %v", err)
	}

	logger.Println("更新 config.json 成功")
}

func deploy_transfer_rsync(ip, username, password string, port int, file string) {
	transfer_directory := fmt.Sprintf("/home/%s/app/transfer", username)
	transfer_rsync_es_file := fmt.Sprintf("/home/%s/app/transfer_rsync_es", username)
	// 创建 SSH 客户端
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	// 连接到远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接到远程服务器: %v", err)
	}
	defer client.Close()

	// 检查transfer目录是否存在
	check_dir_cmd := fmt.Sprintf("test -d %s && echo exists", transfer_directory)
	output, err := runCommand(client, check_dir_cmd)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}
	// 检查transfer_rsync_es目录是否存在
	check_dir_cmd = fmt.Sprintf("test -d %s && echo exists", transfer_rsync_es_file)
	rsync_output, err := runCommand(client, check_dir_cmd)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}
	if string(output) == "exists\n" {
		// 如果目录存在,则执行cp命令
		var copy_cmd string
		if string(rsync_output) == "exists\n" {
			copy_cmd = fmt.Sprintf("cp -r %s/. %s/.", transfer_directory, transfer_rsync_es_file)
		} else {
			copy_cmd = fmt.Sprintf("cp -r %s %s", transfer_directory, transfer_rsync_es_file)
		}
		if _, err := runCommand(client, copy_cmd); err != nil {
			logger.Fatalf("复制transfer_rsync_es目录失败: %v", err)
		}
		// 创建sftp客户端
		sftp_client, err := sftp.NewClient(client)
		if err != nil {
			logger.Fatalf("无法创建sftp客户端: %v", err)
		}
		defer sftp_client.Close()
		// 打开本地文件
		local_file, err := os.Open(file)
		if err != nil {
			logger.Fatalf("无法打开transfer_rsync_es本地配置文件: %v", err)
		}
		defer local_file.Close()
		//  上传文件到远程目录
		remote_file_path := fmt.Sprintf("%s/config/application-remote.properties", transfer_rsync_es_file)
		remote_file, err := sftp_client.Create(remote_file_path)
		if err != nil {
			logger.Fatalf("无法创建远程transfer_rsync_es配置文件: %v", err)
		}
		defer remote_file.Close()
		if _, err := remote_file.ReadFrom(local_file); err != nil {
			logger.Fatalf("文件上传失败: %v", err)
		}
		logger.Println("文件上传成功,并备份")
	} else {
		logger.Fatalf(" %s 目录不存在，无法执行操作。", transfer_directory)
	}

}

func local_deploy_transfer_rsync(username, file string) {
	transfer_directory := fmt.Sprintf("/home/%s/app/transfer", username)
	transfer_rsync_es_file := fmt.Sprintf("/home/%s/app/transfer_rsync_es", username)

	// 检查transfer目录是否存在
	if _, err := os.Stat(transfer_directory); os.IsNotExist(err) {
		logger.Fatalf(" %s 目录不存在，无法执行操作。", transfer_directory)
	}

	// 检查transfer_rsync_es目录是否存在
	_, err := os.Stat(transfer_rsync_es_file)
	if os.IsNotExist(err) {
		// 如果目录不存在，则创建
		if err := os.MkdirAll(transfer_rsync_es_file, 0755); err != nil {
			logger.Fatalf("无法创建目录 %s: %v", transfer_rsync_es_file, err)
		}
	}

	// 执行cp命令
	copy_cmd := fmt.Sprintf("cp -r %s/. %s/.", transfer_directory, transfer_rsync_es_file)
	if err := exec.Command("sh", "-c", copy_cmd).Run(); err != nil {
		logger.Fatalf("复制transfer_rsync_es目录失败: %v", err)
	}

	// 打开本地文件
	local_file, err := os.Open(file)
	if err != nil {
		logger.Fatalf("无法打开transfer_rsync_es本地配置文件: %v", err)
	}
	defer local_file.Close()

	// 上传文件到远程目录
	remote_file_path := fmt.Sprintf("%s/config/application-remote.properties", transfer_rsync_es_file)
	remote_file, err := os.Create(remote_file_path)
	if err != nil {
		logger.Fatalf("无法创建远程文件: %v", err)
	}
	defer remote_file.Close()

	if _, err := io.Copy(remote_file, local_file); err != nil {
		logger.Fatalf("transfer_rsync_es配置文件上传失败: %v", err)
	}

	logger.Println("transfer_rsync_es配置文件上传成功,并备份")
}

func updateDockerCompose(ip, username, password string, port int) {
	// 创建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接到远程服务器: %v", err)
	}
	defer client.Close()

	// 检查目录是否存在
	checkDirCmd := fmt.Sprintf("test -d /home/%s/app/transfer_rsync_es && echo exists", username)

	output, err := runCommand(client, checkDirCmd)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	if string(output) == "exists\n" {
		// 创建目录
		checkDirCmd = fmt.Sprintf("mkdir -p /home/%s/data/transfer_rsync_es", username)
		_, err := runCommand(client, checkDirCmd)
		if err != nil {
			logger.Fatalf("transfer_rsync_es执行合建目录命令失败: %v", err)
		}
		// 创建 SFTP 客户端
		sftpClient, err := sftp.NewClient(client)
		if err != nil {
			logger.Fatalf("无法创建SFTP客户端: %v", err)
		}
		defer sftpClient.Close()

		// 读取 docker-compose 文件
		dockerComposePath := fmt.Sprintf("/home/%s/app/transfer_rsync_es/bin/docker-compose.yml", username)
		tmpDockerComposePath := fmt.Sprintf("/home/%s/app/transfer_rsync_es/bin/docker-compose.yml.tmp", username)

		remoteFile, err := sftpClient.Open(dockerComposePath)
		if err != nil {
			logger.Fatalf("transfer_rsync_es无法打开远程docker-compose文件: %v", err)
		}
		defer remoteFile.Close()

		// 解析 YAML 数据
		var composeData map[string]interface{}
		if err := yaml.NewDecoder(remoteFile).Decode(&composeData); err != nil {
			logger.Fatalf("无法解析 YAML 数据: %v", err)
		}

		// 更新服务名称
		if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
			if transferService, ok := services["transfer"].(map[interface{}]interface{}); ok {
				services["transfer_rsync_es"] = transferService
				delete(services, "transfer")
				if _, ok := transferService["container_name"].(string); ok {
					transferService["container_name"] = "transfer_rsync_es"
				}
			}
		}

		// 更新 volumes
		if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
			if transferService, ok := services["transfer_rsync_es"].(map[interface{}]interface{}); ok {
				if volumes, ok := transferService["volumes"].([]interface{}); ok {
					for _, volume := range volumes {
						if volumeMap, ok := volume.(map[interface{}]interface{}); ok {
							if src, ok := volumeMap["source"].(string); ok {
								if strings.Contains(src, "/app/transfer/log") {
									volumeMap["source"] = strings.Replace(src, "/app/transfer/log", "/app/transfer_rsync_es/log", 1)
								} else if strings.Contains(src, "/data/transfer") {
									volumeMap["source"] = strings.Replace(src, "/data/transfer", "/data/transfer_rsync_es", 1)
								} else if strings.Contains(src, "/app/transfer/config/log4j2.xml") {
									volumeMap["source"] = strings.Replace(src, "/app/transfer/config/log4j2.xml", "/app/transfer_rsync_es/config/log4j2.xml", 1)
								}
							}
						}
					}

					// 添加新的 volume
					newVolume := map[string]interface{}{
						"type":      "bind",
						"source":    fmt.Sprintf("/home/%s/app/transfer_rsync_es/config/application-remote.properties", username),
						"target":    "/root/transfer/config/application-remote.properties",
						"read_only": true,
					}
					volumes = append(volumes, newVolume)
					transferService["volumes"] = volumes
				}

				// 更新端口
				if ports, ok := transferService["ports"].([]interface{}); ok {
					for i, portMapping := range ports {
						if portStr, ok := portMapping.(string); ok && portStr == "15001:15001" {
							ports[i] = "15101:15101"
							break
						}
					}
				}
			}
		}

		// 写入临时文件
		tmpFile, err := sftpClient.Create(tmpDockerComposePath)
		if err != nil {
			logger.Fatalf("无法创建transfer_rsync_es临时文件: %v", err)
		}
		defer tmpFile.Close()

		if err := yaml.NewEncoder(tmpFile).Encode(composeData); err != nil {
			logger.Fatalf("无法写入transfer_rsync_es临时文件: %v", err)
		}

		// 替换原文件
		if err := sftpClient.Remove(dockerComposePath); err != nil {
			logger.Fatalf("无法删除transfer_rsync_es原文件: %v", err)
		}
		if err := sftpClient.Rename(tmpDockerComposePath, dockerComposePath); err != nil {
			logger.Fatalf("无法重命名transfer_rsync_es文件: %v", err)
		}

		logger.Println("成功更新transfer_rsync_es docker-compose.yml")
	} else {
		logger.Fatal("transfer_rsync_es目录不存在，退出")
	}
}

func localUpdateDockerCompose(username string) {
	transferRsyncEsDir := fmt.Sprintf("/home/%s/app/transfer_rsync_es", username)

	// 检查 transfer_rsync_es 目录是否存在
	if _, err := os.Stat(transferRsyncEsDir); os.IsNotExist(err) {
		logger.Fatal("transfer_rsync_es目录不存在，退出")
	}

	// 创建数据目录
	dataDir := fmt.Sprintf("/home/%s/data/transfer_rsync_es", username)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Fatalf("无法创建transfer_rsync_es数据目录: %v", err)
	}

	// 读取 docker-compose 文件
	dockerComposePath := fmt.Sprintf("%s/bin/docker-compose.yml", transferRsyncEsDir)
	tmpDockerComposePath := fmt.Sprintf("%s/bin/docker-compose.yml.tmp", transferRsyncEsDir)

	// 打开文件
	file, err := os.Open(dockerComposePath)
	if err != nil {
		logger.Fatalf("无法打开transfer_rsync_es服务docker-compose文件: %v", err)
	}
	defer file.Close()

	// 解析 YAML 数据
	var composeData map[string]interface{}
	if err := yaml.NewDecoder(file).Decode(&composeData); err != nil {
		logger.Fatalf("无法解析 YAML 数据: %v", err)
	}

	// 更新服务名称
	if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
		if transferService, ok := services["transfer"].(map[interface{}]interface{}); ok {
			services["transfer_rsync_es"] = transferService
			delete(services, "transfer")
			if _, ok := transferService["container_name"].(string); ok {
				transferService["container_name"] = "transfer_rsync_es"
			}
		}
	}

	// 更新 volumes
	if services, ok := composeData["services"].(map[interface{}]interface{}); ok {
		if transferService, ok := services["transfer_rsync_es"].(map[interface{}]interface{}); ok {
			if volumes, ok := transferService["volumes"].([]interface{}); ok {
				for _, volume := range volumes {
					if volumeMap, ok := volume.(map[interface{}]interface{}); ok {
						if src, ok := volumeMap["source"].(string); ok {
							if strings.Contains(src, "/app/transfer/log") {
								volumeMap["source"] = strings.Replace(src, "/app/transfer/log", "/app/transfer_rsync_es/log", 1)
							} else if strings.Contains(src, "/data/transfer") {
								volumeMap["source"] = strings.Replace(src, "/data/transfer", "/data/transfer_rsync_es", 1)
							} else if strings.Contains(src, "/app/transfer/config/log4j2.xml") {
								volumeMap["source"] = strings.Replace(src, "/app/transfer/config/log4j2.xml", "/app/transfer_rsync_es/config/log4j2.xml", 1)
							}
						}
					}
				}

				// 添加新的 volume
				newVolume := map[string]interface{}{
					"type":      "bind",
					"source":    fmt.Sprintf("%s/config/application-remote.properties", transferRsyncEsDir),
					"target":    "/root/transfer/config/application-remote.properties",
					"read_only": true,
				}
				volumes = append(volumes, newVolume)
				transferService["volumes"] = volumes
			}

			// 更新端口
			if ports, ok := transferService["ports"].([]interface{}); ok {
				for i, portMapping := range ports {
					if portStr, ok := portMapping.(string); ok && portStr == "15001:15001" {
						ports[i] = "15101:15101"
						break
					}
				}
			}
		}
	}

	// 如果文件存在则先删除
	if _, err := os.Stat(tmpDockerComposePath); err == nil {
		if err := os.Remove(tmpDockerComposePath); err != nil {
			logger.Fatalf("无法删除transfer_rsync_es已存在的文件: %v", err)
		}
		logger.Printf("已删除transfer_rsync_es已存在的文件: %s", tmpDockerComposePath)
	} else if !os.IsNotExist(err) {
		logger.Fatalf("检查transfer_rsync_es文件是否存在时出错: %v", err)
	}
	// 写入临时文件
	tmpFile, err := os.Create(tmpDockerComposePath)
	if err != nil {
		logger.Fatalf("无法创建transfer_rsync_es临时文件: %v", err)
	}
	defer tmpFile.Close()

	if err := yaml.NewEncoder(tmpFile).Encode(composeData); err != nil {
		logger.Fatalf("无法写入transfer_rsync_es临时文件: %v", err)
	}

	// 替换原文件
	if err := os.Remove(dockerComposePath); err != nil {
		logger.Fatalf("无法删除transfer_rsync_es原文件: %v", err)
	}
	if err := os.Rename(tmpDockerComposePath, dockerComposePath); err != nil {
		logger.Fatalf("无法重命名transfer_rsync_es文件: %v", err)
	}

	logger.Println("成功更新transfer_rsync_es docker-compose.yml")
}

func updateServiceScript(ip, username, password string, port int) {
	// 创建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接到远程服务器: %v", err)
	}
	defer client.Close()

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Fatalf("无法创建 SFTP 客户端: %v", err)
	}
	defer sftpClient.Close()

	// 定义脚本路径
	scriptPath := fmt.Sprintf("/home/%s/app/transfer_rsync_es/bin/service.sh", username)

	// 打开脚本文件
	scriptFile, err := sftpClient.Open(scriptPath)
	if err != nil {
		logger.Fatalf("无法打开transfer_rsync_es service.sh脚本文件: %v", err)
	}
	defer scriptFile.Close()

	// 读取文件内容
	content, err := io.ReadAll(scriptFile)
	if err != nil {
		logger.Fatalf("无法读取transfer_rsync_es service.sh脚本文件: %v", err)
	}

	// 替换内容
	newContent := strings.ReplaceAll(string(content), "transfer", "transfer_rsync_es")

	// 创建临时文件
	tmpFilePath := fmt.Sprintf("/home/%s/app/transfer_rsync_es/bin/service.sh.tmp", username)
	tmpFile, err := sftpClient.Create(tmpFilePath)
	if err != nil {
		logger.Fatalf("无法创建transfer_rsync_es临时文件: %v", err)
	}
	defer tmpFile.Close()

	// 写入新内容
	if _, err := tmpFile.Write([]byte(newContent)); err != nil {
		logger.Fatalf("无法写入transfer_rsync_es临时文件: %v", err)
	}

	// 替换原文件
	if err := sftpClient.Remove(scriptPath); err != nil {
		logger.Fatalf("无法删除transfer_rsync_es原文件: %v", err)
	}
	if err := sftpClient.Rename(tmpFilePath, scriptPath); err != nil {
		logger.Fatalf("无法重命名transfer_rsync_es文件: %v", err)
	}

	logger.Println("成功更新transfer_rsync_es service.sh 脚本")
}

func localUpdateServiceScript(username string) {
	transferRsyncEsDir := fmt.Sprintf("/home/%s/app/transfer_rsync_es", username)

	// 检查 transfer_rsync_es 目录是否存在
	if _, err := os.Stat(transferRsyncEsDir); os.IsNotExist(err) {
		logger.Fatal("transfer_rsync_es 目录不存在，退出")
	}

	// 定义脚本路径
	scriptPath := fmt.Sprintf("%s/bin/service.sh", transferRsyncEsDir)

	// 打开脚本文件
	scriptFile, err := os.Open(scriptPath)
	if err != nil {
		logger.Fatalf("无法打开transfer_rsync_es脚本service.sh文件: %v", err)
	}
	defer scriptFile.Close()

	// 读取文件内容
	content, err := io.ReadAll(scriptFile)
	if err != nil {
		logger.Fatalf("无法读取transfer_rsync_es脚本文件service.sh: %v", err)
	}

	// 替换内容
	newContent := strings.ReplaceAll(string(content), "transfer", "transfer_rsync_es")

	// 创建临时文件
	tmpFilePath := fmt.Sprintf("%s/bin/service.sh.tmp", transferRsyncEsDir)
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		logger.Fatalf("无法创建transfer_rsync_es临时文件: %v", err)
	}
	defer tmpFile.Close()

	// 写入新内容
	if _, err := tmpFile.Write([]byte(newContent)); err != nil {
		logger.Fatalf("无法写入transfer_rsync_es临时文件: %v", err)
	}

	// 替换原文件
	if err := os.Remove(scriptPath); err != nil {
		logger.Fatalf("无法删除transfer_rsync_es原文件: %v", err)
	}
	if err := os.Rename(tmpFilePath, scriptPath); err != nil {
		logger.Fatalf("无法重命名transfer_rsync_es文件: %v", err)
	}

	logger.Println("成功更新transfer_rsync_es service.sh 脚本")
}

func startTransferRsyncEs(ip, username, password string, port int) {
	// 创建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接到远程服务器: %v", err)
	}
	defer client.Close()

	// 检查目录是否存在
	checkDirCmd := fmt.Sprintf("test -d /data/%s/app/transfer_rsync_es && echo exists", username)
	output, err := runCommand(client, checkDirCmd)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	if string(output) == "exists\n" {
		// 启动服务
		startCmd := fmt.Sprintf("source ~/.bash_profile && cd /home/%s/app/transfer_rsync_es/bin && docker-compose up -d", username)
		output, err = runCommand(client, startCmd)
		if err != nil {
			logger.Fatalf("启动transfer_rsync_es服务失败: %v", err)
		}
		logger.Printf("服务transfer_rsync_es启动成功: %s", string(output))
	} else {
		logger.Fatal("transfer_rsync_es 目录不存在")
	}
}

func stopRemoteTransferRsyncEs(ip, username, password string, port int) {
	// 创建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 连接到远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("无法连接到远程服务器: %v", err)
	}
	defer client.Close()

	// 检查目录是否存在
	checkDirCmd := fmt.Sprintf("test -d /data/%s/app/transfer_rsync_es && echo exists", username)
	output, err := runCommand(client, checkDirCmd)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	if string(output) == "exists\n" {
		// 启动服务
		startCmd := fmt.Sprintf("source ~/.bash_profile && cd /home/%s/app/transfer_rsync_es/bin && docker-compose down", username)
		output, err = runCommand(client, startCmd)
		if err != nil {
			logger.Fatalf("停止transfer_rsync_es服务失败: %v", err)
		}
		logger.Printf("transfer_rsync_es服务停止成功: %s", string(output))
	} else {
		logger.Fatal("transfer_rsync_es 目录不存在")
	}
}

func localStartTransferRsyncEs(username string) {
	transferRsyncEsDir := fmt.Sprintf("/data/%s/app/transfer_rsync_es", username)

	// 检查 transfer_rsync_es 目录是否存在
	if _, err := os.Stat(transferRsyncEsDir); os.IsNotExist(err) {
		logger.Fatal("transfer_rsync_es 目录不存在，退出")
	}

	// 启动服务
	startCmd := fmt.Sprintf("source ~/.bash_profile && cd /home/%s/app/transfer_rsync_es/bin && docker-compose up -d", username)
	cmd := exec.Command("sh", "-c", startCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatalf("启动服务失败: %v", err)
	}

	logger.Printf("服务启动成功: %s", string(output))
}

func localStopTransferRsyncEs(username string) {
	transferRsyncEsDir := fmt.Sprintf("/data/%s/app/transfer_rsync_es", username)

	// 检查 transfer_rsync_es 目录是否存在
	if _, err := os.Stat(transferRsyncEsDir); os.IsNotExist(err) {
		logger.Fatal("transfer_rsync_es 目录不存在，退出")
	}

	// 启动服务
	startCmd := fmt.Sprintf("source ~/.bash_profile && cd /home/%s/app/transfer_rsync_es/bin && docker-compose down", username)
	cmd := exec.Command("sh", "-c", startCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatalf("停止服务失败: %v", err)
	}

	logger.Printf("服务停止成功: %s", string(output))
}

func updateHBaseSite(ip, username, password string, port int, hbaseFile string) {

	// XML结构定义
	type Property struct {
		Name  string `xml:"name"`
		Value string `xml:"value"`
	}

	type Configuration struct {
		XMLName  xml.Name   `xml:"configuration"`
		Property []Property `xml:"property"`
	}
	// SSH客户端配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 建立SSH连接
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		logger.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		logger.Fatalf("创建SFTP客户端失败: %v", err)
	}
	defer sftpClient.Close()

	// 读取远程文件
	remoteFile, err := sftpClient.Open(hbaseFile)
	if err != nil {
		logger.Fatalf("打开文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 解析XML内容
	var config Configuration
	decoder := xml.NewDecoder(remoteFile)
	if err := decoder.Decode(&config); err != nil {
		logger.Fatalf("解析XML失败: %v", err)
	}

	// 查找并修改hbase.replication属性
	foundReplication := false
	foundWAL := false
	for i, prop := range config.Property {
		if prop.Name == "hbase.replication" {
			config.Property[i].Value = "true"
			foundReplication = true
		}
		if prop.Name == "wal.enabled" {
			config.Property[i].Value = "true"
			foundWAL = true
		}
		if foundReplication && foundWAL {
			break
		}
	}

	// 如果未找到则添加新属性
	if !foundReplication {
		config.Property = append(config.Property, Property{
			Name:  "hbase.replication",
			Value: "true",
		})
	}
	if !foundWAL {
		config.Property = append(config.Property, Property{
			Name:  "wal.enabled",
			Value: "true",
		})
	}

	// 生成格式化的XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("  ", "    ")
	if err := encoder.Encode(config); err != nil {
		logger.Fatalf("生成XML失败: %v", err)
	}

	// 写入临时文件
	tmpFile := hbaseFile + ".tmp"
	if _, err := sftpClient.Stat(tmpFile); err == nil {
		if err := sftpClient.Remove(tmpFile); err != nil {
			logger.Fatalf("无法删除已存在的临时文件: %v ", err)
		}
	}
	dstFile, err := sftpClient.Create(tmpFile)
	if err != nil {
		logger.Fatalf("创建临时文件失败: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, bytes.NewReader(buf.Bytes())); err != nil {
		logger.Fatalf("写入临时文件失败: %v", err)
	}
	// 先检查目标文件是否存在，如果存在则删除
	if _, err := sftpClient.Stat(hbaseFile); err == nil {
		if err := sftpClient.Remove(hbaseFile); err != nil {
			logger.Fatalf("无法删除已存在的文件: %v", err)
		}
	}
	// 重命名文件
	if err := sftpClient.Rename(tmpFile, hbaseFile); err != nil {
		logger.Fatalf("重命名文件失败: %v", err)
	}

	logger.Println("配置文件更新成功")
}

func LocalUpdateHBaseSite(hbaseFile string) {
	// XML结构定义
	type Property struct {
		Name  string `xml:"name"`
		Value string `xml:"value"`
	}

	type Configuration struct {
		XMLName  xml.Name   `xml:"configuration"`
		Property []Property `xml:"property"`
	}

	// 检查 hbaseFile 文件是否存在
	if _, err := os.Stat(hbaseFile); os.IsNotExist(err) {
		logger.Fatalf("文件 %s 不存在，退出", hbaseFile)
	}

	// 读取文件
	file, err := os.Open(hbaseFile)
	if err != nil {
		logger.Fatalf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 解析XML内容
	var config Configuration
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		logger.Fatalf("解析XML失败: %v", err)
	}

	// 查找并修改 hbase.replication 属性
	// 查找并修改hbase.replication属性
	foundReplication := false
	// foundWAL := false
	sourceNb := false
	sourceSize := false
	sourceRatio := false
	regioWal := false
	replSleep := false
	for i, prop := range config.Property {
		if prop.Name == "hbase.replication" {
			config.Property[i].Value = "true"
			foundReplication = true
		}
		if prop.Name == "replication.source.nb.capacity" {
			config.Property[i].Value = "5000"
			sourceNb = true
		}
		if prop.Name == "replication.source.size.capacity" {
			config.Property[i].Value = "4194304"
			sourceSize = true
		}
		if prop.Name == "replication.source.ratio" {
			config.Property[i].Value = "1"
			sourceRatio = true
		}
		if prop.Name == "hbase.regionserver.wal.enablecompression" {
			config.Property[i].Value = "false"
			regioWal = true
		}
		if prop.Name == "replication.sleep.before.failover" {
			config.Property[i].Value = "5000"
			replSleep = true
		}

		// if prop.Name == "wal.enabled" {
		// 	config.Property[i].Value = "true"
		// 	foundWAL = true
		// }
		if foundReplication && sourceNb && sourceSize && sourceRatio && regioWal && replSleep {
			break
		}
	}

	// 如果未找到则添加新属性
	if !foundReplication {
		config.Property = append(config.Property, Property{
			Name:  "hbase.replication",
			Value: "true",
		})
	}

	if !sourceNb {
		config.Property = append(config.Property, Property{
			Name:  "replication.source.nb.capacity",
			Value: "5000",
		})
	}

	if !sourceSize {
		config.Property = append(config.Property, Property{
			Name:  "replication.source.size.capacity",
			Value: "4194304",
		})
	}

	if !sourceRatio {
		config.Property = append(config.Property, Property{
			Name:  "replication.source.ratio",
			Value: "1",
		})
	}

	if !regioWal {
		config.Property = append(config.Property, Property{
			Name:  "hbase.regionserver.wal.enablecompression",
			Value: "false",
		})
	}

	if !replSleep {
		config.Property = append(config.Property, Property{
			Name:  "replication.sleep",
			Value: "5000",
		})
	}

	// if !foundWAL {
	// 	config.Property = append(config.Property, Property{
	// 		Name:  "wal.enabled",
	// 		Value: "true",
	// 	})
	// }

	// 生成格式化的XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("  ", "    ")
	if err := encoder.Encode(config); err != nil {
		logger.Fatalf("生成XML失败: %v", err)
	}

	// 写入临时文件
	tmpFile := hbaseFile + ".tmp"
	if _, err := os.Stat(tmpFile); err == nil {
		if err := os.Remove(tmpFile); err != nil {
			logger.Fatalf("无法删除已存在的临时文件: %v", err)
		}
	}

	tmpFileHandle, err := os.Create(tmpFile)
	if err != nil {
		logger.Fatalf("创建临时文件失败: %v", err)
	}
	defer tmpFileHandle.Close()

	if _, err := io.Copy(tmpFileHandle, bytes.NewReader(buf.Bytes())); err != nil {
		logger.Fatalf("写入临时文件失败: %v", err)
	}

	// 先检查目标文件是否存在，如果存在则删除
	if _, err := os.Stat(hbaseFile); err == nil {
		if err := os.Remove(hbaseFile); err != nil {
			logger.Fatalf("无法删除已存在的文件: %v", err)
		}
	}

	// 重命名文件
	if err := os.Rename(tmpFile, hbaseFile); err != nil {
		logger.Fatalf("重命名文件失败: %v", err)
	}

	logger.Println("配置文件更新成功")
}

func restartHBase(ip, username, password string, port int) error {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// 检查 HBase 目录是否存在
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)
	checkCmd := fmt.Sprintf("test -d %s && echo exists", hbaseDir)
	output, err := runCommand(conn, checkCmd)
	if err != nil || string(output) != "exists\n" {
		logger.Fatal("HBase 目录不存在")
		return fmt.Errorf("HBase 目录不存在: %s", hbaseDir)
	}

	// 停止 HBase 服务
	stopCmd := fmt.Sprintf("bash /home/%s/server/hbase/bin/service.sh stop", username)
	if _, err := runCommand(conn, stopCmd); err != nil {
		logger.Fatal("停止 HBase 服务失败")
		return fmt.Errorf("停止 HBase 服务失败: %v", err)
	}
	logger.Println("HBase 服务已停止")

	// 启动 HBase 服务
	startCmd := fmt.Sprintf("bash /home/%s/server/hbase/bin/service.sh start", username)
	if _, err := runCommand(conn, startCmd); err != nil {
		return fmt.Errorf("启动 HBase 服务失败: %v", err)
	}
	logger.Println("HBase 服务已启动")

	return nil
}

func localRestartHBase(username string) error {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatal("HBase 目录不存在")
		return fmt.Errorf("HBase 目录不存在: %s", hbaseDir)
	}

	// 停止 HBase 服务
	stopCmd := fmt.Sprintf("bash /home/%s/server/hbase/bin/service.sh stop", username)
	cmd := exec.Command("sh", "-c", stopCmd)
	if err := cmd.Run(); err != nil {
		logger.Fatal("停止 HBase 服务失败")
		return fmt.Errorf("停止 HBase 服务失败: %v", err)
	}
	logger.Println("HBase 服务已停止")

	// 启动 HBase 服务
	startCmd := fmt.Sprintf("bash /home/%s/server/hbase/bin/service.sh start", username)
	cmd = exec.Command("sh", "-c", startCmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动 HBase 服务失败: %v", err)
	}
	logger.Println("HBase 服务已启动")

	return nil
}

func checkHBaseReplication(ip, username, password string, port int, remoteZookeeper string) {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		logger.Fatalf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// // 创建会话
	// session, err := conn.NewSession()
	// if err != nil {
	// 	logger.Fatalf("创建会话失败: %v", err)
	// }
	// defer session.Close()

	// 检查是否已启用 HBase Replication
	command := fmt.Sprintf("echo list_peers | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED", username)
	output, _ := runCommand(conn, command)
	// output, err := runCommand(session, command)

	if strings.Contains(string(output), "ENABLED") {
		logger.Println("已有 HBase Replication 同步，跳过配置")
		return
	}

	logger.Println("未检测到 HBase Replication,开始尝试配置同步...")

	// // 创建会话
	// session, err = conn.NewSession()
	// if err != nil {
	// 	logger.Fatalf("创建会话失败: %v", err)
	// }
	// defer session.Close()
	// 添加新的 Replication Peer
	command = fmt.Sprintf("echo \"add_peer '100', '%s'\" | /home/%s/server/hbase/hbase/bin/hbase shell", remoteZookeeper, username)
	_, err = runCommand(conn, command)
	if err != nil {
		logger.Fatalf("添加 Replication Peer 失败: %v", err)
	}

	// // 创建会话
	// session, err = conn.NewSession()
	// if err != nil {
	// 	logger.Fatalf("创建会话失败: %v", err)
	// }
	// defer session.Close()
	// 再次检查是否成功启用
	command = fmt.Sprintf("echo list_peers | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED", username)
	output, err = runCommand(conn, command)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	if strings.Contains(string(output), "ENABLED") {
		logger.Println("配置 HBase Replication 成功!")
	} else {
		logger.Fatal("配置失败，脚本退出!")
	}
}

func localCheckHBaseReplication(username, remoteZookeeper string) error {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatalf("HBase 目录不存在: %s", hbaseDir)
		return fmt.Errorf("HBase 目录不存在: %s", hbaseDir)
	}

	// 检查是否已启用 HBase Replication
	command := fmt.Sprintf("echo list_peers | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED", username)
	cmd := exec.Command("sh", "-c", command)
	output, _ := cmd.CombinedOutput()

	if strings.Contains(string(output), "ENABLED") {
		logger.Println("已有 HBase Replication 同步，跳过配置")
		return nil
	}

	logger.Println("未检测到 HBase Replication, 开始尝试配置同步...")

	// 添加新的 Replication Peer
	command = fmt.Sprintf("echo \"add_peer '100', '%s'\" | /home/%s/server/hbase/hbase/bin/hbase shell", remoteZookeeper, username)
	cmd = exec.Command("sh", "-c", command)
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Fatalf("添加 Replication Peer 失败: %v", err)
		return fmt.Errorf("添加 Replication Peer 失败: %v", err)
	}

	// 再次检查是否成功启用
	command = fmt.Sprintf("echo list_peers | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep ENABLED", username)
	cmd = exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
		return fmt.Errorf("执行命令失败: %v", err)
	}

	if strings.Contains(string(output), "ENABLED") {
		logger.Println("配置 HBase Replication 成功!")
	} else {
		logger.Fatal("配置失败，脚本退出!")
		return fmt.Errorf("配置失败")
	}

	return nil
}

// runCommand 执行命令并返回输出
func runCommand(conn *ssh.Client, command string) ([]byte, error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %v", err)
	}
	return output, nil
}

func checkHBaseStatus(ip, username, password string, port int) bool {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		logger.Fatalf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// 执行命令检查 HBase 状态
	command := fmt.Sprintf("echo status | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep dead | awk -F ',' '{print $4}' | tr -d ' dead'", username)
	output, err := runCommand(conn, command)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	// 检查输出
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "0" {
		logger.Println("HBase 服务正常")
		return true
	} else {
		logger.Printf("HBase 服务异常，其中有 %s 状态异常\n", output)
		return false
	}
}

func localCheckHBaseStatus(username string) bool {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatalf("HBase 目录不存在: %s", hbaseDir)
		return false
	}

	// 执行命令检查 HBase 状态
	command := fmt.Sprintf("echo status | /home/%s/server/hbase/hbase/bin/hbase shell 2> /dev/null | grep dead | awk -F ',' '{print $4}' | tr -d ' dead'", username)
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
		return false
	}

	// 检查输出
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "0" {
		logger.Println("HBase 服务正常")
		return true
	} else {
		logger.Printf("HBase 服务异常，其中有 %s 状态异常\n", outputStr)
		return false
	}
}

func checkReplicationScope(ip, username, password string, port int) {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		logger.Fatalf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// 创建会话
	session, err := conn.NewSession()
	if err != nil {
		logger.Fatalf("创建会话失败: %v", err)
	}
	defer session.Close()

	// 正则表达式提取 REPLICATION_SCOPE 的值
	re := regexp.MustCompile(`REPLICATION_SCOPE\s*=>\s*'(\d+)'`)

	// 遍历表及其列族
	for table, columnFamilies := range TABLE_FAMILY_MAP {
		for _, cf := range strings.Split(columnFamilies, ",") {
			// 执行命令获取表描述
			command := fmt.Sprintf("echo \"describe '%s'\" | /home/%s/server/hbase/hbase/bin/hbase shell 2>/dev/null", table, username)
			output, err := runCommand(conn, command)
			if err != nil {
				logger.Fatalf("执行命令失败: %v", err)
			}

			// 使用正则表达式提取 REPLICATION_SCOPE 的值
			match := re.FindStringSubmatch(string(output))
			if len(match) > 1 {
				replicationScopeValue := match[1]
				logger.Printf("table: %s, cf: %s, REPLICATION_SCOPE: %s\n", table, cf, replicationScopeValue)
			} else {
				logger.Println("未找到 REPLICATION_SCOPE 或 HBase 服务异常")
			}
		}
	}
}

func localCheckReplicationScope(username string) {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatalf("HBase 目录不存在: %s", hbaseDir)
		return
	}

	// 正则表达式提取 REPLICATION_SCOPE 的值
	re := regexp.MustCompile(`REPLICATION_SCOPE\s*=>\s*'(\d+)'`)

	// 遍历表及其列族
	for table, columnFamilies := range TABLE_FAMILY_MAP {
		for _, cf := range strings.Split(columnFamilies, ",") {
			// 执行命令获取表描述
			command := fmt.Sprintf("echo \"describe '%s'\" | /home/%s/server/hbase/hbase/bin/hbase shell 2>/dev/null", table, username)
			cmd := exec.Command("sh", "-c", command)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Fatalf("执行命令失败: %v", err)
				return
			}

			// 使用正则表达式提取 REPLICATION_SCOPE 的值
			match := re.FindStringSubmatch(string(output))
			if len(match) > 1 {
				replicationScopeValue := match[1]
				logger.Printf("table: %s, cf: %s, REPLICATION_SCOPE: %s\n", table, cf, replicationScopeValue)
			} else {
				logger.Println("未找到 REPLICATION_SCOPE 或 HBase 服务异常")
			}
		}
	}
}

func checkTablesExistence(ip, username, password string, port int) bool {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		logger.Fatalf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// // 创建会话
	// session, err := conn.NewSession()
	// if err != nil {
	// 	logger.Fatalf("创建会话失败: %v", err)
	// }
	// defer session.Close()

	// 执行命令获取表列表
	command := fmt.Sprintf("echo list | /home/%s/server/hbase/hbase/bin/hbase shell 2>/dev/null", username)
	output, err := runCommand(conn, command)
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
	}

	// 遍历 TABLE_FAMILY_MAP 的键，检查是否存在于表列表中
	for key := range TABLE_FAMILY_MAP {
		if strings.Contains(string(output), key) {
			logger.Printf("%s exists in the table list.\n", key)
		} else {
			logger.Printf("%s does not exist in the table list.\n", key)
			return false
		}
	}

	return true
}

func localCheckTablesExistence(username string) bool {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatalf("HBase 目录不存在: %s", hbaseDir)
		return false
	}

	// 执行命令获取表列表
	command := fmt.Sprintf("echo list | /home/%s/server/hbase/hbase/bin/hbase shell 2>/dev/null", username)
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatalf("执行命令失败: %v", err)
		return false
	}

	// 遍历 TABLE_FAMILY_MAP 的键，检查是否存在于表列表中
	for key := range TABLE_FAMILY_MAP {
		if strings.Contains(string(output), key) {
			logger.Printf("%s exists in the table list.\n", key)
		} else {
			logger.Printf("%s does not exist in the table list.\n", key)
			return false
		}
	}

	return true
}

func setTablesFamily(ip, username, password string, port int, value int) {
	// SSH 客户端配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		logger.Fatalf("SSH 连接失败: %v", err)
	}
	defer conn.Close()

	// 创建会话
	session, err := conn.NewSession()
	if err != nil {
		logger.Fatalf("创建会话失败: %v", err)
	}
	defer session.Close()

	// HBase shell 路径
	hbaseShellPath := fmt.Sprintf("/home/%s/server/hbase/hbase/bin/hbase shell", username)

	// 遍历表及其列族
	for table, columnFamilies := range TABLE_FAMILY_MAP {
		for _, cf := range strings.Split(columnFamilies, ",") {
			// 构建要执行的命令
			commands := fmt.Sprintf(`
				disable '%s';
				alter '%s', {NAME => '%s', REPLICATION_SCOPE => '%d'};
				enable '%s';
				`, table, table, cf, value, table)

			// 使用 SSH 执行命令
			command := fmt.Sprintf("echo \"%s\" | %s 2>/dev/null", commands, hbaseShellPath)
			output, err := runCommand(conn, command)
			if err != nil {
				logger.Printf("执行命令失败: %v\n", err)
			}

			// 记录输出和错误
			if string(output) != "" {
				logger.Printf("Output for table %s, cf %s: %s\n", table, cf, output)
			}
		}
	}
}

func localSetTablesFamily(username string, value int) {
	hbaseDir := fmt.Sprintf("/home/%s/server/hbase", username)

	// 检查 HBase 目录是否存在
	if _, err := os.Stat(hbaseDir); os.IsNotExist(err) {
		logger.Fatalf("HBase 目录不存在: %s", hbaseDir)
		return
	}

	// HBase shell 路径
	hbaseShellPath := fmt.Sprintf("/home/%s/server/hbase/hbase/bin/hbase shell", username)

	// 遍历表及其列族
	for table, columnFamilies := range TABLE_FAMILY_MAP {
		for _, cf := range strings.Split(columnFamilies, ",") {
			// 构建要执行的命令
			commands := fmt.Sprintf(`
				disable '%s';
				alter '%s', {NAME => '%s', REPLICATION_SCOPE => '%d'};
				enable '%s';
				`, table, table, cf, value, table)

			// 使用本地命令执行
			command := fmt.Sprintf("echo \"%s\" | %s 2>/dev/null", commands, hbaseShellPath)
			cmd := exec.Command("sh", "-c", command)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Printf("执行命令失败: %v\n", err)
			}

			// 记录输出和错误
			if string(output) != "" {
				logger.Printf("Output for table %s, cf %s: %s\n", table, cf, output)
			}
		}
	}
}

func editShake(v *viper.Viper) {
	// 获取当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		logger.Fatalf("获取当前目录失败: %v", err)
	}

	// 构建 shake.toml 文件路径
	tomlPath := filepath.Join(currentDir, "file", "redis", "config", "shake.toml")

	// 读取 TOML 文件
	var tomlConfig map[string]interface{}
	if _, err := toml.DecodeFile(tomlPath, &tomlConfig); err != nil {
		logger.Fatal(err)
	}
	if err != nil {
		logger.Fatalf("读取 TOML 文件失败: %v", err)
	}
	var readClusterTag bool
	var writeClusterTag bool
	var readAddress string
	var writeAddress string
	var readPassword string
	var writePassword string
	if v.GetString("database.redis.active_mode") == "single" {
		readClusterTag = false
		readAddress = fmt.Sprintf("%s:%d", v.GetString("database.redis.single.spring.redis.host"), v.GetInt("database.redis.single.spring.redis.port"))
		readPassword = v.GetString("database.redis.single.spring.redis.password")
	} else {
		readClusterTag = true
		readAddress = v.GetString("database.redis.cluster.spring.redis.cluster.nodes")
		readPassword = v.GetString("database.redis.cluster.spring.redis.password")
	}
	if v.GetString("database.backup_redis.active_mode") == "single" {
		writeClusterTag = false
		writeAddress = fmt.Sprintf("%s:%d", v.GetString("database.backup_redis.single.spring.redis.host"), v.GetInt("database.backup_redis.single.spring.redis.port"))
		writePassword = v.GetString("database.backup_redis.single.spring.redis.password")
	} else {
		writeClusterTag = true
		writeAddress = v.GetString("database.backup_redis.cluster.spring.redis.cluster.nodes")
		writePassword = v.GetString("database.backup_redis.cluster.spring.redis.password")
	}

	// 获取源 Redis 配置
	syncReader := map[string]interface{}{
		"cluster":  readClusterTag,
		"address":  readAddress,
		"password": readPassword,
	}

	// 获取备份 Redis 配置
	redisWriter := map[string]interface{}{
		"cluster":  writeClusterTag,
		"address":  writeAddress,
		"password": writePassword,
	}

	// 更新 TOML 配置
	// config.Set("sync_reader", syncReader)
	// config.Set("redis_writer", redisWriter)
	tomlConfig["sync_reader"] = syncReader
	tomlConfig["redis_writer"] = redisWriter

	// 将更新后的配置写入临时文件
	tempFilePath := tomlPath + ".temp"
	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(tomlConfig)
	if err != nil {
		logger.Fatalf("编码 TOML 配置失败: %v", err)
	}

	err = os.WriteFile(tempFilePath, buf.Bytes(), 0644)
	if err != nil {
		logger.Fatalf("写入临时文件失败: %v", err)
	}

	// // 删除原始配置文件
	// err = os.Remove(tomlPath)
	// if err != nil {
	// 	logger.Fatalf("删除原始文件失败: %v", err)
	// }

	// // 将临时文件重命名为原始文件名
	// err = os.Rename(tempFilePath, tomlPath)
	// if err != nil {
	// 	logger.Fatalf("重命名文件失败: %v", err)
	// }
}

func deployRedisShake(ip, username, password string, port int) error {
	// 创建 SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote server: %v", err)
	}
	defer client.Close()

	// 获取 CPU 架构
	cpuArchitecture := runtime.GOARCH
	var redisShakePath string
	switch cpuArchitecture {
	case "amd64":
		redisShakePath = "redis-shake-v4.3.2-linux-amd64.tar.gz"
	case "arm64":
		redisShakePath = "redis-shake-v4.3.2-linux-arm64.tar.gz"
	default:
		return errors.New("unsupported CPU architecture")
	}

	// 创建目录
	commands := []string{
		fmt.Sprintf("mkdir -p /home/%s/ops/redis-shake/bin", username),
		fmt.Sprintf("mkdir -p /home/%s/ops/redis-shake/logs", username),
		fmt.Sprintf("mkdir -p /home/%s/ops/redis-shake/config", username),
	}
	for _, cmd := range commands {
		if _, err := runCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to execute command: %v", err)
		}
	}

	// 上传文件
	scriptDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	redisShakeFile := filepath.Join(scriptDir, "file/redis", redisShakePath)
	redisServiceFile := filepath.Join(scriptDir, "file/redis/config/service.sh")
	redisTomlFile := filepath.Join(scriptDir, "file/redis/config/shake.toml.temp")

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	files := map[string]string{
		redisShakeFile:   fmt.Sprintf("/home/%s/ops/redis-shake/bin/%s", username, redisShakePath),
		redisServiceFile: fmt.Sprintf("/home/%s/ops/redis-shake/bin/service.sh", username),
		redisTomlFile:    fmt.Sprintf("/home/%s/ops/redis-shake/config/shake.toml", username),
	}
	for src, dst := range files {
		srcFile, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("failed to open source file: %v", err)
		}
		defer srcFile.Close()
		dstFile, err := sftpClient.Create(dst)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %v", err)
		}
		defer dstFile.Close()
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy file: %v", err)
		}
	}
	script := fmt.Sprintf(`
		cd /home/%s/ops/redis-shake/bin
		tar zxf %s
		rm -f %s
		chmod +x redis-shake
		rm -f shake.toml
		chmod +x ./service.sh
		bash ./service.sh nodogstart
	`, username, redisShakePath, redisShakePath)
	if _, err := runCommand(client, script); err != nil {
		logger.Fatal(err)
		return err
	}

	return nil
}

func stopRemoteRedisShake(ip, username, password string, port int) error {
	// 创建 SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to remote server: %v", err)
	}
	defer client.Close()
	script := fmt.Sprintf(`
		cd /home/%s/ops/redis-shake/bin
		bash ./service.sh stop
	`, username)
	if _, err := runCommand(client, script); err != nil {
		logger.Fatalf("redis-shake服务停止失败: %v", err)
		return err
	}

	return nil
}

func localDeployRedisShake(username string) error {
	// 获取当前工作目录
	redisPath := fmt.Sprintf("/home/%s/", username)
	scriptDir, err := os.Getwd()
	if err != nil {
		logger.Fatal("Failed to get current working directory:", err)
		return err
	}

	// 获取 CPU 架构
	cpuArchitecture := runtime.GOARCH
	var redisShakePath string
	switch cpuArchitecture {
	case "amd64":
		redisShakePath = "redis-shake-v4.3.2-linux-amd64.tar.gz"
	case "arm64":
		redisShakePath = "redis-shake-v4.3.2-linux-arm64.tar.gz"
	default:
		return errors.New("unsupported CPU architecture")
	}

	// 创建目录
	dirs := []string{
		filepath.Join(redisPath, "ops/redis-shake/bin"),
		filepath.Join(redisPath, "ops/redis-shake/logs"),
		filepath.Join(redisPath, "ops/redis-shake/config"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// 文件路径
	redisShakeFile := filepath.Join(scriptDir, "file/redis", redisShakePath)
	redisServiceFile := filepath.Join(scriptDir, "file/redis/config/service.sh")
	redisTomlFile := filepath.Join(scriptDir, "file/redis/config/shake.toml.temp")

	// 复制 Redis Shake 文件
	srcFile, err := os.Open(redisShakeFile)
	if err != nil {
		return fmt.Errorf("failed to open Redis Shake file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Join(redisPath, "ops/redis-shake/bin", redisShakePath))
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy Redis Shake file: %v", err)
	}

	// 复制 service.sh 文件
	srcFile, err = os.Open(redisServiceFile)
	if err != nil {
		return fmt.Errorf("failed to open service.sh file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err = os.Create(filepath.Join(redisPath, "ops/redis-shake/bin/service.sh"))
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy service.sh file: %v", err)
	}

	// 复制 shake.toml 文件
	srcFile, err = os.Open(redisTomlFile)
	if err != nil {
		return fmt.Errorf("failed to open shake.toml file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err = os.Create(filepath.Join(redisPath, "ops/redis-shake/config/shake.toml"))
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy shake.toml file: %v", err)
	}

	// 执行解压和启动脚本
	binDir := filepath.Join(redisPath, "ops/redis-shake/bin")
	script := fmt.Sprintf(`
        cd %s
        tar zxf %s
        rm -f %s
        chmod +x redis-shake
        chmod +x ./service.sh
        bash ./service.sh nodogstart
    `, binDir, redisShakePath, redisShakePath)

	cmd := exec.Command("bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute script: %v", err)
	}

	return nil
}

func localStopRedisShake(username string) error {
	// 获取当前工作目录
	redisPath := fmt.Sprintf("/home/%s/", username)

	// 执行解压和启动脚本
	binDir := filepath.Join(redisPath, "ops/redis-shake/bin")
	script := fmt.Sprintf(`
        cd %s
        bash ./service.sh stop
    `, binDir)

	cmd := exec.Command("bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("redis-shake服务停止失败: %v", err)
	}

	return nil
}

func remoteEditHosts(ip, username, password string, port int) error {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		log.Fatalf("failed to connect to remote server: %v", err)
		return err
	}
	defer client.Close()

	// 需要注释的域名列表
	domainsToComment := []string{
		"nginx", "redis", "crash", "postgres", "zookeeper", "hbase", "kafka",
		"elasticsearchMaster", "elasticsearchClient", "nebula", "kibana", "init",
		"receiver", "cleaner", "transfer", "threat", "web-service", "analyzer-dev",
		"security-event", "app-sender",
	}

	// 构造sed命令来注释这些域名
	sedCommand := "sudo sed -i "
	for _, domain := range domainsToComment {
		sedCommand += fmt.Sprintf("-e '/%s/s/^/# /' ", domain)
	}
	sedCommand += "/etc/hosts"

	if _, err := runCommand(client, sedCommand); err != nil {
		logger.Fatalf("failed to run sed command: %v", err)
		return err
	}

	// 输出结果
	logger.Printf("Command executed successfully.")
	return nil
}

func remoteAddHosts(ip, username, password string, port int, hostsPath string) error {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接远程服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), sshConfig)
	if err != nil {
		log.Fatalf("failed to connect to remote server: %v", err)
		return err
	}
	defer client.Close()
	// 读取hosts_path文件内容
	linesToAdd, err := readLines(hostsPath)
	if err != nil {
		logger.Fatalf("Failed to read hosts_path file: %v", err)
	}
	// 读取远程服务器的/etc/hosts文件内容
	remoteHostsPath := "/etc/hosts"
	existingLines, err := readRemoteFile(client, remoteHostsPath)
	if err != nil {
		logger.Fatalf("Failed to read remote /etc/hosts file: %v", err)
	}

	// 检查并追加缺失的行
	if err := remoteAppendMissingLines(client, linesToAdd, existingLines); err != nil {
		logger.Fatalf("Failed to update remote /etc/hosts: %v", err)
	}

	log.Println("Remote /etc/hosts updated successfully.")
	return nil
}

// 读取远程文件内容并返回行切片
func readRemoteFile(client *ssh.Client, filePath string) ([]string, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	// 使用cat命令读取文件内容
	if err := session.Run(fmt.Sprintf("cat %s", filePath)); err != nil {
		return nil, fmt.Errorf("failed to read remote file: %v", err)
	}

	// 将输出按行分割
	var lines []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// 检查并追加缺失的行到远程文件
func remoteAppendMissingLines(client *ssh.Client, linesToAdd, existingLines []string) error {
	for _, line := range linesToAdd {
		if !contains(existingLines, line) {
			// 如果行不存在，则追加到远程文件
			command := fmt.Sprintf("echo '%s' | sudo tee -a /etc/hosts", line)
			if _, err := runCommand(client, command); err != nil {
				return fmt.Errorf("failed to append line to remote file: %v", err)
			}
			logger.Printf("Added line: %s", line)
		} else {
			logger.Printf("Line already exists, skipping: %s", line)
		}
	}
	return nil
}

func localEditHosts() error {
	// 需要注释的域名列表
	domainsToComment := []string{
		"nginx", "redis", "crash", "postgres", "zookeeper", "hbase", "kafka",
		"elasticsearchMaster", "elasticsearchClient", "nebula", "kibana", "init",
		"receiver", "cleaner", "transfer", "threat", "web-service", "analyzer-dev",
		"security-event", "app-sender",
	}

	// 构造sed命令来注释这些域名
	sedCommand := "sudo sed -i "
	for _, domain := range domainsToComment {
		sedCommand += fmt.Sprintf("-e '/%s/s/^/# /' ", domain)
	}
	sedCommand += "/etc/hosts"

	// 执行sed命令
	cmd := exec.Command("sh", "-c", sedCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run sed command: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Command executed successfully. Output: %s", string(output))
	return nil
}

// 读取文件内容并返回行切片
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// 检查并追加缺失的行
func appendMissingLines(etcHostsPath string, linesToAdd, existingLines []string) error {
	// 打开/etc/hosts文件以追加内容
	file, err := os.OpenFile(etcHostsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 检查每一行是否需要追加
	for _, line := range linesToAdd {
		if !contains(existingLines, line) {
			if _, err := file.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("failed to write to /etc/hosts: %v", err)
			}
			log.Printf("Added line: %s", line)
		} else {
			log.Printf("Line already exists, skipping: %s", line)
		}
	}
	return nil
}

// 检查切片中是否包含某一行
func contains(lines []string, target string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func setupLogger(localFilePath string) *log.Logger {
	// 创建日志文件路径
	logFilePath := filepath.Join(filepath.Dir(localFilePath), "update_everisk.log")

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}

	// 创建多输出日志器
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	return logger
}

func main() {
	current_dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取脚本当前目录失败")
		return
	}
	config_file_name := "config.yaml"
	config_file_path := filepath.Join(current_dir, "config", config_file_name)
	v := viper.New()
	v.AddConfigPath(current_dir)
	v.SetConfigFile(config_file_path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("配置文件读取错误:", err)
		return
	}
	log_file_path := filepath.Join(current_dir, "update_everisk.log")
	logger = setupLogger(log_file_path)
	logger.Println("日志文件已创建")
	username := v.GetString("database.ssh.username")
	password := v.GetString("database.ssh.password")
	port := v.GetInt("database.ssh.port")

	// 创建命令行参数解析器
	updateInit := flag.Bool("update_init", false, "根据config.yam配置中的内容,更新config.json中的配置")
	updateTransfer := flag.Bool("update_transfer", false, "根据config.yam配置中的内容,复制~/app/transfer到~/app/transfer_rsync_es，并自动配置docker-compose.yaml的内容 及配置文件 ，然后启动")
	updateHbase := flag.Bool("update_hbase", false, "根据config.yam配置中的内容,配置hbase的hbase.replication,然后自动配置hbase中的scope,并推送至远程服务器")
	updateRedis := flag.Bool("update_redis", false, "根据config.yam配置中的内容,启动一个redis_shake,将主的redis实时同步至备的redis")
	stopTransferRsyncEs := flag.Bool("stop_transferRsyncEs", false, "根据config.yam配置中的内容,停止transfer_rsync_es服务")
	stopRedisShake := flag.Bool("stop_redisShake", false, "根据config.yam配置中的内容,停止redis_shake")
	stopHbaseScope := flag.Bool("stop_hbaseScope", false, "根据config.yam配置中的内容,自动配置hbase中的scope为0")
	editHosts := flag.Bool("edit_hosts", false, "需要用root权限，或者sudo免密码权限，将template中的hosts追加到/etc/hosts中，此操作要在更新hbase之前完成")
	// 设置自定义的 Usage 函数
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "威胁感知更新配置，实现威胁灾备功能\n\n")
		fmt.Fprintf(os.Stderr, "使用方法: %s [参数]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "可用参数:\n")
		flag.PrintDefaults()
	}

	// 解析命令行参数
	flag.Parse()

	// 如果没有提供任何参数，打印帮助信息
	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	islocal := v.GetBool("database.local.islocal")
	if !islocal {
		// 根据参数执行相应的模块
		if *updateInit {
			fmt.Println("执行 update_init 模块")
			// 调用 update_init 模块的逻辑
			for _, list := range v.Get("database.init").([]interface{}) {
				if list.(map[string]interface{})["enable"].(bool) {
					ip := list.(map[string]interface{})["ip"].(string)
					ssh_init_everisk_ha(ip, username, password, port, v)
				}
			}
		}
		if *updateTransfer {
			fmt.Println("执行 update_transfer 模块")
			// 调用 update_transfer 模块的逻辑
			for _, list := range v.Get("database.transfer_rsync_es_list").([]interface{}) {
				if list.(map[string]interface{})["enable"].(bool) {
					ip := list.(map[string]interface{})["ip"].(string)
					file := filepath.Join(current_dir, "template/transfer/application-remote.properties")
					deploy_transfer_rsync(ip, username, password, port, file)
					updateDockerCompose(ip, username, password, port)
					updateServiceScript(ip, username, password, port)
					startTransferRsyncEs(ip, username, password, port)
				}
			}
		}
		if *updateHbase {
			fmt.Println("执行 update_hbase 模块")
			// 调用 update_hbase 模块的逻辑
			hbaseRemoteFile := fmt.Sprintf("/home/%s/server/hbase/hbase/conf/hbase-site.xml", username)
			initHbaseRemoateFile := fmt.Sprintf("/home/%s/app/init/config/everisk/hadoop/hbase-site.xml", username)
			if v.GetBool("database.hbase.enable") {
				if hbaseList := v.Get("database.hbase.list"); hbaseList != nil {
					if list, ok := hbaseList.([]interface{}); ok {
						for _, item := range list {
							if ip, ok := item.(string); ok {
								updateHBaseSite(ip, username, password, port, hbaseRemoteFile)
								restartHBase(ip, username, password, port)
								time.Sleep(time.Second * 10)
								for {
									if checkHBaseStatus(ip, username, password, port) {
										break
									}
									time.Sleep(10 * time.Second)
								}
							}
						}
					}
				} else {
					logger.Fatalf("database.hbase.list 配置项类型错误，期望 []string,实际 %T", hbaseList)
				}

			} else {
				logger.Fatalf("未找到 database.hbase.list 配置项")
			}
			for _, list := range v.Get("database.init").([]interface{}) {
				if list.(map[string]interface{})["enable"].(bool) {
					ip := list.(map[string]interface{})["ip"].(string)
					updateHBaseSite(ip, username, password, port, initHbaseRemoateFile)
				}
			}
			if v.GetBool("database.remote_zookeeper.enable") {
				remote_zookeeper := v.GetString("database.remote_zookeeper.zookeeper.server")
				// 获取第一个值
				var hbase_ip string
				if hbaseList := v.Get("database.hbase.list"); hbaseList != nil {
					if list, ok := hbaseList.([]interface{}); ok {
						for _, item := range list {
							if ip, ok := item.(string); ok {
								hbase_ip = ip
								break // 只需要第一个值
							}
						}
					}
				}
				checkHBaseReplication(hbase_ip, username, password, port, remote_zookeeper+":/hbase")
				hbase_status := checkHBaseStatus(hbase_ip, username, password, port)
				// hbase_existence := checkTablesExistence(hbase_ip, username, password, port)
				checkTablesExistence(hbase_ip, username, password, port)
				// if hbase_status && hbase_existence {
				if hbase_status {
					checkReplicationScope(hbase_ip, username, password, port)
					setTablesFamily(hbase_ip, username, password, port, 1)
					checkReplicationScope(hbase_ip, username, password, port)

				}

			}
		}
		if *updateRedis {
			fmt.Println("执行 update_redis 模块")
			// 调用 update_redis 模块的逻辑
			editShake(v)
			if v.GetBool("database.redis_sync.enable") {
				ip := v.GetString("database.redis_sync.ip")
				deployRedisShake(ip, username, password, port)
			}
		}
		if *stopTransferRsyncEs {
			fmt.Println("执行 stop_transfer_rsync_es 模块")
			for _, list := range v.Get("database.transfer_rsync_es_list").([]interface{}) {
				ip := list.(map[string]interface{})["ip"].(string)
				stopRemoteTransferRsyncEs(ip, username, password, port)
			}
		}

		if *stopRedisShake {
			fmt.Println("执行 stop_redis_shake 模块")
			ip := v.GetString("database.redis_sync.ip")
			stopRemoteRedisShake(ip, username, password, port)
		}

		if *stopHbaseScope {
			fmt.Println("执行 stop_hbase_scope 模块")
			// 获取第一个值
			var hbase_ip string
			if hbaseList := v.Get("database.hbase.list"); hbaseList != nil {
				if list, ok := hbaseList.([]interface{}); ok {
					for _, item := range list {
						if ip, ok := item.(string); ok {
							hbase_ip = ip
							break // 只需要第一个值
						}
					}
				}
			}
			hbase_status := checkHBaseStatus(hbase_ip, username, password, port)
			// hbase_existence := checkTablesExistence(hbase_ip, username, password, port)
			checkTablesExistence(hbase_ip, username, password, port)
			// if hbase_status && hbase_existence {
			if hbase_status {
				checkReplicationScope(hbase_ip, username, password, port)
				setTablesFamily(hbase_ip, username, password, port, 0)
				checkReplicationScope(hbase_ip, username, password, port)
			}
		}
		if *editHosts {
			fmt.Println("执行 edit_hosts 模块")
			fmt.Print("是否已经修改template/hosts内容？(yes/y继续，其它退出): ")
			var input string
			fmt.Scanln(&input)
			if strings.ToLower(input) != "yes" && strings.ToLower(input) != "y" {
				logger.Fatalf("未确认修改template/hosts内容，程序退出")
				return
			}
			hostsPath := filepath.Join(current_dir, "template/hosts")
			if v.GetBool("database.hbase.enable") {
				if hbaseList := v.Get("database.hbase.list"); hbaseList != nil {
					if list, ok := hbaseList.([]interface{}); ok {
						for _, item := range list {
							if ip, ok := item.(string); ok {
								remoteEditHosts(ip, username, password, port)
								remoteAddHosts(ip, username, password, port, hostsPath)
							}
						}
					}
				} else {
					logger.Fatalf("database.hbase.list 配置项类型错误，期望 []string,实际 %T", hbaseList)
				}

			}
		}
	} else {
		fmt.Println("执行本机/本地逻辑")
		// 根据参数执行相应的模块
		if *updateInit {
			fmt.Println("执行 update_init 模块")
			initPath := fmt.Sprintf("/home/%s/app/init", username)
			_, err := os.Stat(initPath)
			if err == nil {
				local_ssh_init_everisk_ha(username, v)
			}
		}
		if *updateTransfer {
			fmt.Println("执行 update_transfer 模块")
			transferPath := fmt.Sprintf("/home/%s/app/transfer", username)
			file := filepath.Join(current_dir, "template/transfer/application-remote.properties")
			_, err := os.Stat(transferPath)
			if err == nil {
				local_deploy_transfer_rsync(username, file)
				localUpdateDockerCompose(username)
				localUpdateServiceScript(username)
				localStartTransferRsyncEs(username)
			}
		}
		if *updateHbase {
			fmt.Println("执行 update_hbase 模块")
			// 调用 update_hbase 模块的逻辑
			hbaseRemoteFile := fmt.Sprintf("/home/%s/server/hbase/hbase/conf/hbase-site.xml", username)
			initHbaseRemoateFile := fmt.Sprintf("/home/%s/app/init/config/everisk/hadoop/hbase-site.xml", username)
			if _, err := os.Stat(hbaseRemoteFile); err == nil {
				LocalUpdateHBaseSite(hbaseRemoteFile)
				localRestartHBase(username)
				time.Sleep(time.Second * 10)
				for {
					if localCheckHBaseStatus(username) {
						break
					}
					time.Sleep(10 * time.Second)
				}
				remote_zookeeper := v.GetString("database.remote_zookeeper.zookeeper.server")
				localCheckHBaseReplication(username, remote_zookeeper)
				hbase_status := localCheckHBaseStatus(username)
				// hbase_existence := localCheckTablesExistence(username)
				localCheckTablesExistence(username)
				// if hbase_status && hbase_existence {
				if hbase_status {
					localCheckReplicationScope(username)
					localSetTablesFamily(username, 1)
					localCheckReplicationScope(username)

				}
			}
			if _, err := os.Stat(initHbaseRemoateFile); err == nil {
				LocalUpdateHBaseSite(initHbaseRemoateFile)
			}
		}
		if *updateRedis {
			fmt.Println("执行 update_redis 模块")
			// 调用 update_redis 模块的逻辑
			editShake(v)
			localDeployRedisShake(username)
		}
		if *stopTransferRsyncEs {
			fmt.Println("执行 stop_transfer_rsync_es 模块")
			// 调用 stop_transfer_rsync_es 模块的逻辑
			localStopTransferRsyncEs(username)
		}
		if *stopRedisShake {
			fmt.Println("执行 stop_redis_shake 模块")
			localStopRedisShake(username)
		}
		if *stopHbaseScope {
			fmt.Println("执行 start_hbase_scope 模块")
			hbase_status := localCheckHBaseStatus(username)
			// hbase_existence := localCheckTablesExistence(username)
			localCheckTablesExistence(username)
			// if hbase_status && hbase_existence {
			if hbase_status {
				localCheckReplicationScope(username)
				localSetTablesFamily(username, 0)
				localCheckReplicationScope(username)

			}
		}
		if *editHosts {
			fmt.Println("执行 edit_hosts 模块")
			fmt.Print("是否已经修改template/hosts内容？(yes/y继续，其它退出): ")
			var input string
			fmt.Scanln(&input)
			if strings.ToLower(input) != "yes" && strings.ToLower(input) != "y" {
				logger.Fatalf("未确认修改template/hosts内容，程序退出")
				return
			}
			// 调用 edit_hosts 模块的逻辑
			localEditHosts()
			hostsPath := filepath.Join(current_dir, "template/hosts")
			// 读取hosts_path文件内容
			linesToAdd, err := readLines(hostsPath)
			if err != nil {
				logger.Fatalf("Failed to read hosts_path file: %v", err)
			}

			// 读取/etc/hosts文件内容
			etcHostsPath := "/etc/hosts"
			existingLines, err := readLines(etcHostsPath)
			if err != nil {
				logger.Fatalf("Failed to read /etc/hosts file: %v", err)
			}

			// 检查并追加内容
			if err := appendMissingLines(etcHostsPath, linesToAdd, existingLines); err != nil {
				logger.Fatalf("Failed to update /etc/hosts: %v", err)
			}
		}
	}

}
