/*
 * @Author: 
 * @Date: 2023-07-16 16:20:31
 * @LastEditors: 
 * @LastEditTime: 2023-07-17 10:43:55
 * @FilePath: /go/src/go_code/zstack/login/login.go
 * @Description:
 * Copyright (c) 2023 by , All Rights Reserved.
 */
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type ApiLogInMsg struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
}

type RequestData struct {
	ApiLogInMsg ApiLogInMsg `json:"org.zstack.header.identity.APILogInByAccountMsg"`
}

// 登陆
func main() {
	url := "http://172.16.44.100:8080/zstack/api"
	headers_str := `{ "Content-Type": "application/json" }`
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	if err != nil {
		fmt.Println("headers is ERROR: ", err)
		return
	}
	data_str := ` {
        "org.zstack.header.identity.APILogInByAccountMsg": {
        "password": "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86",
        "accountName": "admin"
        }
    }`
	var data map[string]interface{}
	err = json.Unmarshal([]byte(data_str), &data)
	if err != nil {
		fmt.Println("data is ERROR: ", err)
		return
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(data_str)))
	if err != nil {
		fmt.Println("req is ERROR: ", err)
		return
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value.(string))
	}

	// 发起请求
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("resp is ERROR: ", err)
		return
	}

	fmt.Println("resp is: ", resp)

	defer resp.Body.Close()

	// 解数据
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		fmt.Println("response is ERROR: ", err)
		return
	}
	fmt.Println("response is: ", response)
	result := response["result"]
	fmt.Println("result is: ", result)
	var result_data map[string]interface{}
	err = json.Unmarshal([]byte(result.(string)), &result_data)
	if err != nil {
		fmt.Println("result_data is ERROR: ", err)
		return
	}
	// fmt.Println("result_data is: ", result_data)
	// fmt.Println("result_data is: ", result_data["org.zstack.header.identity.APILogInReply"])
	inventory, ok := result_data["org.zstack.header.identity.APILogInReply"]
	if !ok {
		fmt.Println("inventory is ERROR: ", err)
		return
	}
	fmt.Println("inventory is: ", inventory)
	// uuid_data, ok := inventory.(map[string]interface{})
	uuid_data, ok := inventory.(map[string]interface{})
	fmt.Println("uuid_data is: ", uuid_data)
	// 查看 uuid_data 类型
	fmt.Println("uuid_data type is: ", reflect.TypeOf(uuid_data))
	if !ok {
		fmt.Println("类型断言失败")
		return
	}
	uuid := uuid_data["inventory"].(map[string]interface{})["userUuid"]
	fmt.Println("uuid is: ", uuid)
}
