/*
 * @Author: magician
 * @Date: 2024-07-01 21:56:02
 * @LastEditors: magician
 * @LastEditTime: 2024-07-01 22:03:35
 * @FilePath: /go/src/go_code/zookeeper/zookeeper.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

func check_zookeeper_service(servers []string) bool {
	// 连接到zookeeper集群

	conn, _, err := zk.Connect(servers, 5*time.Second)
	if err != nil {
		fmt.Printf("连接zookeeper失败: %v\n", err)
		return false
	}
	defer conn.Close()

	// 设置连接成功的标志
	state := conn.State()
	if state == zk.StateConnected || state == zk.StateHasSession {
		return true
	}

	fmt.Printf("zookeeper服务状态异常: %v\n", state)
	return false
}

func main() {
	servers := []string{"172.16.51.110:2181"}
	if check_zookeeper_service(servers) {
		fmt.Println("zookeeper服务正常")
	} else {
		fmt.Println("zookeeper服务异常")
	}
}
