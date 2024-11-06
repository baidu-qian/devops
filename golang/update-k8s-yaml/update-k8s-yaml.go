/*
 * @Author: magician
 * @Date: 2024-11-04 13:23:56
 * @LastEditors: magician
 * @LastEditTime: 2024-11-05 13:52:30
 * @FilePath: /go/src/go_code/api替换/api.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func replace_in_files(directory string) ([]string, error) {
	/**
	@brief 查找目录下的yaml文件
	@param directory string 目录路径
	@return []string 文件路径
	**/
	// yamls := []string
	var yamls []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // 如果有错误，返回
		}
		// 检查文件是否以 .yaml 或 .yml 结尾，并且不以 service.yaml 结尾
		if (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) && !strings.HasSuffix(info.Name(), "service.yaml") {
			yamls = append(yamls, path) // 将符合条件的文件路径添加到切片
		}
		return nil // 返回 nil 表示继续遍历
	})

	if err != nil {
		fmt.Println("Error walking the path:", err)
		return yamls, err
	}

	return yamls, nil
}

func update_yaml_variable(file_path, variable_name, new_value string) {
	/**
		@brief 修改yaml文件的环境变量配置
		@param file_path string 文件路径
		@param variable_name string 环境变量名称
		@param new_value string 环境变量值
	**/
	content, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatal(err)
	}
	// Unmarshal YAML data
	var composeData map[string]interface{}
	if err := yaml.Unmarshal(content, &composeData); err != nil {
		log.Fatalf("Failed to unmarshal YAML: %s", err)
	}
	if composeData["kind"].(string) == "StatefulSet" || composeData["kind"].(string) == "Deployment" || composeData["kind"].(string) == "Job" {
		// 更新 'replicas' 字段
		if spec, ok := composeData["spec"].(map[interface{}]interface{}); ok {
			if template, ok := spec["template"].(map[interface{}]interface{}); ok {
				if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
					if containers, ok := podSpec["containers"].([]interface{}); ok { // 这里使用切片类型断言
						for _, container := range containers {
							if containerMap, ok := container.(map[interface{}]interface{}); ok { // 断言为 map
								if envs, ok := containerMap["env"].([]interface{}); ok { // 获取 env 列表
									for _, env := range envs {
										if envMap, ok := env.(map[interface{}]interface{}); ok { // 断言为 map
											if name, exists := envMap["name"]; exists {
												if nameStr, ok := name.(string); ok { // 确保 name 是字符串
													// fmt.Printf("Environment variable name: %s\n", nameStr)

													// 如果需要更新某个环境变量的值，可以在这里进行操作
													if nameStr == variable_name {
														// 更新 value 的值
														envMap["value"] = new_value
														// fmt.Printf("Updated %s to %s\n", nameStr, new_value)
													}
												} else {
													fmt.Println("The value of 'name' is not a string.")
												}
											} else {
												fmt.Println("'name' key does not exist in envMap.")
											}
										}
									}

								}
							}
						}
					}
				}
			}
		}
	}
	updateData, err := yaml.Marshal(composeData)
	if err != nil {
		log.Fatal("Failed to marshal YAML: ", err)
	}
	err = ioutil.WriteFile(file_path, updateData, os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

}

func replace_line_with_placeholder(file_path, new_value string) {
	/*
		@brief 修改文件的mysql连接库数据
		@param file_path string 文件路径
		@param new_value string 连接地址
	**/
	// 打开文件进行读取
	file, err := os.Open(file_path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 使用 bufio.Scanner 逐行读取
	var lines []string
	scanner := bufio.NewScanner(file)

	// 定义正则表达式
	re := regexp.MustCompile(`^\s*(?:db\.url\.0=jdbc:mysql|db\.url\.0=jdbc:postgresql)`)

	for scanner.Scan() {
		line := scanner.Text()

		// 检查当前行是否匹配正则表达式
		if re.MatchString(line) {
			lines = append(lines, new_value) // 替换为占位符
		} else {
			lines = append(lines, line) // 保持原行
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// 打开文件进行写入
	outputFile, err := os.Create(file_path)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer outputFile.Close()

	// 将修改后的内容写回文件
	for _, line := range lines {
		_, err := outputFile.WriteString(line + "\n")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
}

func update_yaml_images(file_path, image_name string) {
	/*
		@brief 修改yaml文件的镜像
		@param file_path string 文件路径
		@param image_name string 镜像名称
	**/
	content, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatal(err)
	}
	// Unmarshal YAML data
	var composeData map[string]interface{}
	if err := yaml.Unmarshal(content, &composeData); err != nil {
		log.Fatalf("Failed to unmarshal YAML: %s", err)
	}
	if composeData["kind"].(string) == "StatefulSet" || composeData["kind"].(string) == "Deployment" || composeData["kind"].(string) == "Job" {
		// 更新 'replicas' 字段
		if spec, ok := composeData["spec"].(map[interface{}]interface{}); ok {
			if template, ok := spec["template"].(map[interface{}]interface{}); ok {
				if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
					if containers, ok := podSpec["containers"].([]interface{}); ok { // 这里使用切片类型断言
						for _, container := range containers {
							if containerMap, ok := container.(map[interface{}]interface{}); ok { // 断言为 map
								if image, ok := containerMap["image"].(string); ok { // 获取 image 列表
									lastColonIndex := strings.LastIndex(image, ":")
									if lastColonIndex != -1 {
										// 分割镜像名称和标签
										imageParts := []string{
											image[:lastColonIndex],   // 镜像名称部分
											image[lastColonIndex+1:], // 标签部分
										}

										// 提取镜像名称（去掉注册表部分）
										parts := strings.Split(imageParts[0], "/")
										imageNameOnly := parts[len(parts)-1] // 获取最后一个部分

										// 构建新的镜像地址
										newImage := fmt.Sprintf("%s/%s:%s", image_name, imageNameOnly, imageParts[1])
										containerMap["image"] = newImage
									}
								} else {
									fmt.Println("Image key does not exist or is not a string.")
								}
							}
						}
					}
				}
			}
		}
	}
	updateData, err := yaml.Marshal(composeData)
	if err != nil {
		log.Fatal("Failed to marshal YAML: ", err)
	}
	err = ioutil.WriteFile(file_path, updateData, os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
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
	redis_host := v.GetStringMapString("database.redis")["host"]
	redis_port := v.GetStringMapString("database.redis")["port"]
	redis_str := fmt.Sprintf("%s:%s", redis_host, redis_port)
	// redis_password := v.GetStringMapString("database.redis")["password"]
	mysql_ip := v.GetStringMapString("database.mysql")["ip"]
	mysql_port := v.GetStringMapString("database.mysql")["port"]
	mysql_username := v.GetStringMapString("database.mysql")["username"]
	mysql_password := v.GetStringMapString("database.mysql")["password"]
	mysql_url := fmt.Sprintf("%s:%s@tcp(%s:%s)/api-sec?charset=utf8&parseTime=True&loc=Local", mysql_username, mysql_password, mysql_ip, mysql_port)
	mysql_url_jdbc := fmt.Sprintf("    db.url.0=jdbc:mysql://%s:%s/nacos?${MYSQL_SERVICE_DB_PARAM:characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false&allowPublicKeyRetrieval=true}", mysql_ip, mysql_port)
	clickhouse_ip := v.GetStringMapString("database.clickhouse")["ip"]
	clickhouse_port := v.GetStringMapString("database.clickhouse")["port"]
	clickhouse_username := v.GetStringMapString("database.clickhouse")["username"]
	clickhouse_password := v.GetStringMapString("database.clickhouse")["password"]
	clickhouse_url := fmt.Sprintf("clickhouse://%s:%s@%s:%s/default", clickhouse_username, clickhouse_password, clickhouse_ip, clickhouse_port)
	// current_year_directory := '运维/api替换/api-k8s-deploy'
	current_year_directory := filepath.Join(current_dir, "api-k8s-deploy")
	image_path := v.GetStringMapString("database.image")["path"]
	yamls, err := replace_in_files(current_year_directory)
	if err != nil {
		fmt.Println("查找yaml配置文件失败")
		fmt.Println(err)
	}
	for _, i := range yamls {
		update_yaml_variable(i, "REDIS_ENDPOINTS", redis_str)
		update_yaml_variable(i, "KAFKA_ENDPOINTS", kafka_brokers)
		update_yaml_variable(i, "MYSQL_URL", mysql_url)
		update_yaml_variable(i, "MYSQL_SERVICE_HOST", mysql_ip)
		update_yaml_variable(i, "MYSQL_SERVICE_PORT", mysql_port)
		update_yaml_variable(i, "MYSQL_SERVICE_USER", mysql_username)
		update_yaml_variable(i, "MYSQL_SERVICE_PASSWORD", mysql_password)
		update_yaml_variable(i, "MYSQL_ADDRESS", mysql_ip+":"+mysql_port)
		update_yaml_variable(i, "MYSQL_USER", mysql_username)
		update_yaml_variable(i, "MYSQL_PASSWORD", mysql_password)
		update_yaml_variable(i, "CLICKHOUSE_URL", clickhouse_url)
		update_yaml_variable(i, "CLICKHOUSE_ADDRESS", clickhouse_ip+":"+clickhouse_port)
		update_yaml_variable(i, "CLICKHOUSE_USER", clickhouse_username)
		update_yaml_variable(i, "CLICKHOUSE_PASSWORD", clickhouse_password)
		replace_line_with_placeholder(i, mysql_url_jdbc)
		update_yaml_images(i, image_path)
	}
}
