/*
 * @Author: magician
 * @Date: 2024-08-11 21:36:51
 * @LastEditors: magician
 * @LastEditTime: 2024-08-12 00:00:11
 * @FilePath: /go/src/go_code/weixin-http/weixin-http.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

type ServerInfo struct {
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	CurrentTime string `json:"current_time"`
}

func getServerINfo() ServerInfo {
	hostname, _ := os.Hostname()
	addrs, err := net.LookupHost(hostname)
	ipAddress := "Unknown"
	if err == nil && len(addrs) > 0 {
		ipAddress = addrs[0]
	}
	currentTime := time.Now().Format(time.RFC3339)

	return ServerInfo{
		Hostname:    hostname,
		IPAddress:   ipAddress,
		CurrentTime: currentTime,
	}
}

func serverInfo(w http.ResponseWriter, r *http.Request) {
	info := getServerINfo()
	jsonResponse, _ := json.Marshal(info)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func main() {
	http.HandleFunc("/status", serverInfo)
	fmt.Println("starting  server on :8080....")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
