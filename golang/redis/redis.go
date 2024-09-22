/*
 * @Author: magician
 * @Date: 2024-06-25 01:03:13
 * @LastEditors: magician
 * @LastEditTime: 2024-06-28 00:38:31
 * @FilePath: /go/src/go_code/redis/redis.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"context"
	"fmt"

	// "github.com/go-redis/redis"
	"github.com/redis/go-redis"
)

func check_redis_status(addr string, password string, db int) bool {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	pong, err := client.Ping(context.Background()).Result()
	if err != nil || pong != "PONG" {
		return false
	}
	return true
}

func main() {
	ip := "172.16.xx.xx"
	port := 6379
	addr := fmt.Sprintf("%s:%d", ip, port)
	password := "123456"
	db := 0
	if check_redis_status(addr, password, db) {
		fmt.Println("redis连接正常")
	} else {
		fmt.Println("redis连接异常")
	}

}
