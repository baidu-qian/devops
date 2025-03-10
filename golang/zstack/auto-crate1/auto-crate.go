/*
 * @Author: magician
 * @Date: 2023-07-28 00:18:34
 * @LastEditors: magician
 * @LastEditTime: 2023-08-03 01:50:53
 * @FilePath: /go/src/go_code/zstack/auto-crate1/auto-crate.go
 * @Description:
 * Copyright (c) 2023 by magician, All Rights Reserved.
 */
package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type APIUserADMIN struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
}

type APILogin struct {
	APIUserADMIN APIUserADMIN `json:"logInByAccount"`
}

// get请求通用接口
func api_get_call(url string, headers map[string]string) (map[string]interface{}, error) {
	// 创建client，超时5s
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
		req.Header.Set(key, value)
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

// put请求通用接口
func api_put_call(url string, headers map[string]interface{}, data map[string]interface{}) (string, error) {
	// 创建client，超时5s
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	json_data, err := json.Marshal(data)
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
	job_uuid := response["inventory"].(map[string]interface{})["uuid"].(string)
	var query_until_done func() string
	query_until_done = func() string {
		// 创建GET请求
		req, err := http.NewRequest("GET", url+"/result/"+job_uuid, nil)
		if err != nil {
			time.Sleep(1 * time.Second)
			fmt.Printf("job[uuid:%s] is still in processing\n", job_uuid)
			return query_until_done()
		}
		// 设置请求头
		for key, value := range headers {
			req.Header.Set(key, value.(string))
		}
		// 发起请求
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(1 * time.Second)
			fmt.Printf("job[uuid:%s] is still in processing\n", job_uuid)
			return query_until_done()
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			time.Sleep(1 * time.Second)
			fmt.Printf("job[uuid:%s] is still in processing\n", job_uuid)
			return query_until_done()
		}
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			time.Sleep(1 * time.Second)
			fmt.Printf("job[uuid:%s] is still in processing\n", job_uuid)
			return query_until_done()
		}
		return response["inventory"].(map[string]interface{})["uuid"].(string)

	}
	//  调用 query_until_done()
	return query_until_done(), nil
}

func login() {
	url := "http://172.16.44.100:8080/zstack/v1/accounts/login"
	headers_str := `{
        "Content-Type": "application/json;charset=UTF-8"
    }`
	var headers map[string]interface{}
	err := json.Unmarshal([]byte(headers_str), &headers)
	// data := map[string]interface{}{
	//     "logInByAccount": map[string]interface{}{
	//         "accountName": "admin",
	//         "password": "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86",
	// }
	// }
	data_str := `{
        "logInByAccount": {
            "accountName": "admin",
            "password": "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86"
        }
    }`
	// var data APILogin
	var data map[string]interface{}
	err = json.Unmarshal([]byte(data_str), &data)
	session_uuid, err := api_put_call(url, headers, data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(session_uuid)
}

// 返回sha512_hash加密值
func sha512_hash(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	return hashString
}

func main() {
	// cpum := flag.String("cpum", "1C1G", "cpu/内存的值 ")
	// image := flag.String("image", "template-Centos7.9-admin", "镜像的值")
	// l3 := flag.String("l3", "admin-44", "l3网络的值")
	// flag.Parse()

	// // 调用main函数，并传入参数
	// mainFunc(*cpum, *image, *l3)
	// fmt.Println(sha512_hash("password"))
	login()
}

// func mainFunc(cpum string, image string, l3 string) {
// 	fmt.Println("cpum: ", cpum)
// 	fmt.Println("image: ", image)
// 	fmt.Println("l3: ", l3)
// }
