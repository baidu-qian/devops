/*
 * @Author: magician
 * @Date: 2024-08-22 22:13:40
 * @LastEditors: magician
 * @LastEditTime: 2024-08-22 23:20:04
 * @FilePath: /go/src/go_code/expect/2/expect.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() {
	// 获取所有的命令行参数
	args := os.Args[1:]

	// 检查是否有参数传递
	if len(args) == 0 {
		fmt.Println("请追加expect要执行的命令")
		return
	}

	// command := fmt.Sprintf("\"%s\"", strings.Join(args, " "))
	// command := "passwd test"

	// 执行命令
	cmd := exec.Command(args[0])
	cmd.Args = append(cmd.Args, args[1:]...)
	fmt.Println(cmd)

	// 获取命令的输入管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Error getting StdinPipe:", err)
		return
	}

	// 启动命令
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	defer stdin.Close()

	// 启动一个 goroutine 来处理输入
	go func() {
		// 等待一段时间以确保命令提示出现
		time.Sleep(2 * time.Second) // 根据实际情况调整时间

		// 向命令的输入管道中写入 "yes"
		_, err := io.WriteString(stdin, "yes\n")
		if err != nil {
			fmt.Println("expect参数错误:", err)
			return
		}

		// // 向命令的输入管道中写入密码
		// _, err = io.WriteString(stdin, "beap123\n")
		// if err != nil {
		// 	fmt.Println("密码输入错误:", err)
		// 	return
		// }
	}()

	// 等待命令执行完成
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return
	}

	fmt.Println("redis cluster初始化完成")
}
