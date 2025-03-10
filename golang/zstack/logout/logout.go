/*
 * @Author: magician
 * @Date: 2023-07-16 11:06:27
 * @LastEditors: magician
 * @LastEditTime: 2023-07-16 11:48:37
 * @FilePath: /go/src/go_code/zstack/logout/logout.go
 * @Description:
 * Copyright (c) 2023 by magician, All Rights Reserved.
 */
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type APILogOutMsg struct {
	SessionUUID string `json:"sessionUuid"`
}

type RequestData struct {
	APILogOutMsg APILogOutMsg `json:"org.zstack.header.identity.APILogOutMsg"`
}

// 退出登录
func main() {
	// 定义url
	url := "http://172.16.44.100:8080/zstack/api"
	headers_str := `{ "Content-Type": "application/json" }`
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonData := `
	{
		"org.zstack.header.identity.APILogOutMsg": {
			"sessionUuid": "b366a81093e144bc8a992420b5b26a62"
		}
	}
	`
	var data RequestData
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		fmt.Println(err)
		return
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		fmt.Println(err)
		return
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value.(string))
		fmt.Println(key, value)
	}
	// 反馈结果
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to make HTTP request: %s\n", err)
		return
	}
	defer resp.Body.Close()
	// 解感数据
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Response 数值为: ", response)
	// 获取result数据
	result_str := response["result"].(string)
	fmt.Println("Response result 数值: ", result_str)
	var result_data map[string]interface{}
	err = json.Unmarshal([]byte(result_str), &result_data)
	if err != nil {
		fmt.Println("result_data 解消失败:", err)
		return
	}

	// 获取success数据
	success := result_data["org.zstack.header.identity.APILogOutReply"].(map[string]interface{})["success"].(bool)
	fmt.Println("success 数値为: ", success)

}
