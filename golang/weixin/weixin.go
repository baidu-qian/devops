/*
 * @Author: hongchun.you
 * @Date: 2023-10-24 22:11:34
 * @LastEditors: magician
 * @LastEditTime: 2024-07-23 11:08:35
 * @FilePath: /go/src/go_code/es/weixin/weixin.go
 * @Description:
 * Copyright (c) 2023 by hongchun.you, All Rights Reserved.
 */
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/go-elasticsearch"
	"github.com/go-zookeeper/zk"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
)

// indices_count is a function that returns the count of indices in Elasticsearch.
// It takes the URL, indices, username, and password as input parameters.
// It returns the count as a string and any error that occurred.
func indices_count(url, indices, username, password string) (string, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{url},
		Username:  username,
		Password:  password,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return "", err
	}
	res, err := es.Cat.Count(
		es.Cat.Count.WithIndex(indices),
		es.Cat.Count.WithPretty(),
	)
	if err != nil || res.StatusCode != 200 {
		return "", fmt.Errorf("ES服务连接失败!")
	}
	lines := strings.Split(res.String(), "\n")
	if len(lines) > 1 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 3 {
			count := parts[len(parts)-1]
			return count, nil
		}
	}
	return "", err
}

// validate_index is a function that validates an index by comparing the total count before and after a sleep period.
// It takes the URL, username, password, sleep time, and index name as input parameters.
// If the final total count is greater than the initial total count, it returns the index name.
// Otherwise, it returns an empty string and any error that occurred.
func validate_index(url, username, password string, sleep_time time.Duration, index_name string) (string, error) {
	initial_total_count, err := indices_count(url, index_name, username, password)
	fmt.Println("es:count:", index_name+":"+initial_total_count)
	if err != nil {
		return "", err
	}
	time.Sleep(sleep_time)
	final_total_count, err := indices_count(url, index_name, username, password)
	fmt.Println("es:final_count:", index_name+":"+final_total_count)
	if err != nil {
		return "", err
	}
	if final_total_count > initial_total_count {
		return "", nil
	} else {
		return index_name, nil
	}
}

// send_md is a function that sends a markdown message to a webhook.
// It takes the webhook URL and content as input parameters.
// It returns any error that occurred.
func send_md(webhook, content string) error {
	header_str := `{
		"Content-Type": "application/json",
        "Charset": "UTF-8"
	}`
	var header map[string]interface{}
	err := json.Unmarshal([]byte(header_str), &header)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(json_data))
	if err != nil {
		return err
	}
	for key, value := range header {
		req.Header.Set(key, value.(string))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return err
	}
	return nil
}

// api_get_call is a function that makes a GET API call with headers and basic authentication.
// It takes the URL, headers, and authentication information as input parameters.
// It returns the response data as a map[string]interface{} and any error that occurred.
func api_get_call(url string, headers map[string]interface{}, auth *BasicAuth) (map[string]interface{}, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value.(string))
	}
	req.SetBasicAuth(auth.Username, auth.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// return nil,  err
		// 返回nil和错误
		return nil, fmt.Errorf("请求失败")

	}
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}
	return responseData, nil
}

type ignoreSSLTransport struct{}

func (t *ignoreSSLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr := &http.Transport{}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return tr.RoundTrip(req)
}

func api_get_call_nginx(url string) (bool, error) {
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &ignoreSSLTransport{},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// return nil,  err
		// 返回nil和错误
		return false, fmt.Errorf("请求失败")

	}
	return true, nil
}

// BasicAuth contains basic authentication information.
type BasicAuth struct {
	Username string
	Password string
}

// ServiceStatus contains service status information.
type ServiceStatus struct {
	IP     string
	Port   string
	Status string
}

// check_service_status is a function that checks the status of a service.
// It takes the Viper configuration, headers, and service name as input parameters.
// It returns a list of services with non-UP status and any error that occurred.
func check_service_status(v *viper.Viper, header map[string]interface{}, service string) ([]string, error) {
	service_list := v.Get("database." + service).([]interface{})
	status_list := []string{}
	auth := &BasicAuth{
		Username: v.GetStringMapString("database.prometheus")["username"],
		Password: v.GetStringMapString("database.prometheus")["password"],
	}
	for _, app := range service_list {
		app_map, ok := app.(map[string]interface{})
		if !ok {
			continue
		}
		ip, ipOk := app_map["ip"].(string)
		port, portOk := app_map["port"].(int)
		if !ipOk || !portOk {
			continue
		}
		enable, _ := app_map["enable"].(bool)
		if !enable {
			continue
		}
		url := fmt.Sprintf("http://%s:%d/actuator/health", ip, port)
		// app_status, err := api_get_call(url, header, auth)
		// if err != nil || app_status["status"].(string) != "UP" {
		// 	fmt.Println(service + ":" + ip + "异常")
		// 	status_list = append(status_list, service+":"+ip)
		// 	return status_list, err
		// }
		// Check for UP status 3 times
		isUp := false
		for i := 0; i < 3; i++ {
			// time.Sleep(5 * time.Second) // Sleep for 5 seconds between checks
			app_status, err := api_get_call(url, header, auth)
			if err != nil || app_status["status"].(string) != "UP" {
				time.Sleep(5 * time.Second) // Sleep for 5 seconds between checks
				// fmt.Println(service + ":" + ip + "异常")
				if i == 2 {
					isUp = false // 如果在最后一次检查中状态仍然不是 "UP"，则设置标志为 false
				}
			} else {
				isUp = true // 如果检测到 "UP" 状态，设置标志为 true 并退出循环
				break
			}
		}
		if !isUp {
			// fmt.Println(service + ":" + ip + "异常")
			status_list = append(status_list, service+":"+ip)
		}
	}
	return status_list, nil
}

// check_all_services 是一个检查所有服务状态的函数。
// 它接受 Viper 配置作为输入参数。
// 它返回一个包含非 UP 状态的服务列表和任何发生的错误。
func check_all_services(v *viper.Viper) ([]string, error) {
	header_str := `{
		"Content-Type": "application/json",
		"Charset": "UTF-8"
	}`
	var header map[string]interface{}
	err := json.Unmarshal([]byte(header_str), &header)
	if err != nil {
		return nil, err
	}
	services_to_check := []string{"receiver", "cleaner", "security-event", "threat", "threat-index", "transfer", "web-service"}
	app_status_list := []string{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, service := range services_to_check {
		wg.Add(1)
		go func(service string) {
			defer wg.Done()
			status_list, _ := check_service_status(v, header, service)
			mu.Lock()
			app_status_list = append(app_status_list, status_list...)
			mu.Unlock()
		}(service)
	}
	wg.Wait()
	return app_status_list, nil
}

// 连接kafka，获取不同group的offset
func get_kafka_lag(brokerList []string) (map[string]map[string]int64, error) {
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
	groupIds, err := admin.ListConsumerGroups()
	if err != nil {
		return nil, err
	}

	// for groupId, _ := range groupIds {
	// 	fmt.Println(groupId)
	// }

	type Result struct {
		groupId string
		lag     map[string]int64
		err     error
	}

	lags := make(map[string]map[string]int64)
	results := make(chan Result, len(groupIds))

	for groupId := range groupIds {
		go func(groupId string) {
			groupLag := make(map[string]int64)
			groups, err := admin.ListConsumerGroupOffsets(groupId, nil)
			if err != nil {
				results <- Result{groupId, nil, err}
				return
			}
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
			results <- Result{groupId, groupLag, nil}
		}(groupId)
	}

	for range groupIds {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		lags[result.groupId] = result.lag
	}

	return lags, nil
}

func check_postgres(db *sql.DB) bool {
	result, err := db.Query("SELECT 1")
	if err != nil {
		return false
	}
	defer result.Close()

	var value int
	for result.Next() {
		err := result.Scan(&value)
		if err != nil {
			return false
		}
	}

	if value == 1 {
		return true
	} else {
		return false
	}
}

func get_postgres_list(v *viper.Viper) ([]string, error) {
	// pg_msg := []string{}
	service_list := v.Get("database.postgres").([]interface{})
	status_list := []string{}
	for _, pg := range service_list {
		pg_map, ok := pg.(map[string]interface{})
		if !ok {
			continue
		}
		ip, ipOk := pg_map["ip"].(string)
		port, portOk := pg_map["port"].(int)
		database, dbOk := pg_map["database"].(string)
		username, unOk := pg_map["username"].(string)
		password, pwOk := pg_map["password"].(string)
		enable, enOk := pg_map["enable"].(bool)
		if !ipOk || !portOk || !dbOk || !unOk || !pwOk || !enOk {
			continue
		}
		if !enable {
			continue
		}
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", ip, port, username, password, database)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			fmt.Println("Error opening database:", err)
			status_list = append(status_list, ip)
		}
		defer db.Close()
		if check_postgres(db) {
			fmt.Println("postgres is working correctly.")
		} else {
			fmt.Println("postgres is not working correctly.")
			status_list = append(status_list, ip)
		}
	}
	return status_list, nil
}

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

func check_zookeeper_status(servers []string) bool {
	// 连接到zookeeper集群

	conn, _, err := zk.Connect(servers, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	time.Sleep(5 * time.Second)

	// 设置连接成功的标志
	state := conn.State()
	if state == zk.StateConnected || state == zk.StateHasSession {
		return true
	}

	return false
}

// send_email_msg sends an email message using the specified parameters.
//
// Parameters:
//   - to: the email address of the recipient
//   - from: the email address of the sender
//   - subject: the subject of the email
//   - content: the content of the email
//   - smtp: the SMTP server address
//   - port: the port number for the SMTP server
//   - login_name: the login name for authentication
//   - login_pass: the login password for authentication
//
// Return type: error
func send_email_msg(to []interface{}, from string, subject string, content string, smtp string, port int, login_name string, login_pass string) error {
	m := gomail.NewMessage()
	//发送人
	m.SetHeader("From", from)
	// //接收人
	// send := []string{}
	// for _, list := range to {
	// 	send = append(send, list.(string))
	// }
	// m.SetHeader("To", strings.Join(send, ","))
	//抄送人
	//m.SetAddressHeader("Cc", "xxx@qq.com", "xiaozhujiao")
	//主题
	m.SetHeader("Subject", subject)
	//内容
	m.SetBody("text/plain", content)
	//附件
	//m.Attach("./myIpPic.png")

	//拿到token，并进行连接,第4个参数是填授权码
	d := gomail.NewDialer(smtp, port, login_name, login_pass)

	// 发送邮件
	for _, list := range to {
		m.SetHeader("To", list.(string))
		if err := d.DialAndSend(m); err != nil {
			return err
			fmt.Println("Error sending email:", err)
		}
	}
	// if err := d.DialAndSend(m); err != nil {
	// 	return err
	// }
	return nil
}

// background_task is a function that performs background tasks.
// It takes the Viper configuration and sleep time as input parameters.
func background_task(v *viper.Viper, sleep_time time.Duration) {
	username := v.GetStringMapString("database.es")["username"]
	password := v.GetStringMapString("database.es")["password"]
	url := v.GetStringMapString("database.es")["url"]
	kafka_brokers := v.GetStringMapString("database.kafka")["brokers"]
	kafka_lag := v.GetStringMapString("database.kafka")["lag"]
	app_list := []string{"app", "dev_status", "devinfo", "env", "msg", "start", "threat", "user_data", "app"}
	results := []string{}
	es_tag := true
	if v.GetBool("database.es.enable") {
		var wg sync.WaitGroup
		for i, v := range app_list {
			wg.Add(1)
			go func(i int, v string) {
				defer wg.Done()
				indices_name := "bb_i_" + v + "*"
				index_name, err := validate_index(url, username, password, sleep_time, indices_name)
				if index_name != "" && err == nil {
					results = append(results, index_name)
				}
				if err != nil {
					es_tag = false
				}
			}(i, v)
		}
		wg.Wait()
	}
	//
	// var kafka_wg sync.WaitGroup
	// for i, v := range app_list {
	// 	wg.Add(1)
	// 	go func(i int, v string) {
	// 		defer wg.Done()
	// 		lags, err := get_kafka_lag(strings.Split(kafka_brokers, ","))
	// 		kafka_lag_int, _ := strconv.ParseInt(kafka_lag, 10, 64)
	// 		if err != nil {
	// 			fmt.Println("获取kafka lag失败")
	// 			return
	// 		}
	// 	}(i, v)
	// }
	// kafka_wg.Wait()
	// lags, err := get_kafka_lag(strings.Split(kafka_brokers, ","))
	// kafka_lag_int, _ := strconv.ParseInt(kafka_lag, 10, 64)
	// if err != nil {
	// 	fmt.Println("获取kafka lag失败")
	// 	return
	// }
	//
	kafka_msg := []string{}
	if v.GetBool("database.kafka.enable") {
		lags, err := get_kafka_lag(strings.Split(kafka_brokers, ","))
		kafka_lag_int, _ := strconv.ParseInt(kafka_lag, 10, 64)
		if err != nil {
			// fmt.Println("获取kafka lag失败")
			kafka_msg = append(kafka_msg, fmt.Sprintf("获取kafka lag失败: %v", err))
		}
		for groupId, topics := range lags {
			for topic, lag := range topics {
				if lag > kafka_lag_int {
					kafka_msg = append(kafka_msg, fmt.Sprintf("kafka groupid:%s, topic:%s, lag:%d", groupId, topic, lag))
				}
			}
		}
	}

	app_msg, _ := check_all_services(v)
	send_weixin := ""
	if len(results) != 0 {
		send_msg := "# 威胁感知ES告警: \n"
		for index, value := range results {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知es表:<font color=\"warning\">%s</font>最近5分钟未增长! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	if !es_tag {
		send_weixin = send_weixin + "# 威胁感知ES告警: \n"
		send_weixin = send_weixin + fmt.Sprintf("1. %s ES服务异常, 请检查! \n", url)
	}
	if len(app_msg) != 0 {
		send_msg := "# 威胁感知APP告警: \n"
		for index, value := range app_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知app:<font color=\"warning\">%s</font>当前状态不为UP! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	if len(kafka_msg) != 0 {
		send_msg := "# 威胁感知kafka告警: \n"
		for index, value := range kafka_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知kafka:<font color=\"warning\">%s</font>当前有堆积! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	// postgres检测
	pg_msg, _ := get_postgres_list(v)
	if len(pg_msg) != 0 {
		send_msg := "# 威胁感知postgres告警: \n"
		for index, value := range pg_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知postgres:<font color=\"warning\">%s</font>当前异常! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	// redis检查
	service_list := v.Get("database.redis").([]interface{})
	redis_msg := []string{}
	for _, redis := range service_list {
		redis_map, ok := redis.(map[string]interface{})
		if !ok {
			continue
		}
		addr := fmt.Sprintf("%s:%d", redis_map["ip"].(string), redis_map["port"].(int))
		enable, _ := redis_map["enable"].(bool)
		if !enable {
			continue
		}
		redis_status := check_redis_status(addr, redis_map["password"].(string), 0)
		if !redis_status {
			redis_msg = append(redis_msg, addr)
		}
	}
	if len(redis_msg) != 0 {
		send_msg := "# 威胁感知redis告警: \n"
		for index, value := range redis_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知redis:<font color=\"warning\">%s</font>当前异常! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}

	// zookeeper检查
	zookeeper_list := v.Get("database.zookeeper").([]interface{})
	zookeeper_msg := []string{}
	for _, zookeeper := range zookeeper_list {
		zookeeper_map, ok := zookeeper.(map[string]interface{})
		if !ok {
			continue
		}
		addr := []string{fmt.Sprintf("%s:%d", zookeeper_map["ip"].(string), zookeeper_map["port"].(int))}
		enable, _ := zookeeper_map["enable"].(bool)
		if !enable {
			continue
		}
		zookeeper_status := check_zookeeper_status(addr)
		if !zookeeper_status {
			zookeeper_msg = append(zookeeper_msg, zookeeper_map["ip"].(string))
		}
	}

	if len(zookeeper_msg) != 0 {
		send_msg := "# 威胁感知zookeeper告警: \n"
		for index, value := range zookeeper_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知zookeeper:<font color=\"warning\">%s</font>当前异常! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	// hbase检查
	hbase_list := v.Get("database.hbase").([]interface{})
	hbase_msg := []string{}
	for _, hbase := range hbase_list {
		hbase_map, ok := hbase.(map[string]interface{})
		if !ok {
			continue
		}
		addr := fmt.Sprintf("http://%s:%d", hbase_map["ip"].(string), hbase_map["port"].(int))
		enable, _ := hbase_map["enable"].(bool)
		if !enable {
			continue
		}
		hbase_status, _ := api_get_call_nginx(addr)
		if !hbase_status {
			hbase_msg = append(hbase_msg, addr)
		}
	}
	if len(hbase_msg) != 0 {
		send_msg := "# 威胁感知hbase告警: \n"
		for index, value := range hbase_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知hbase:<font color=\"warning\">%s</font>当前异常! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}
	// nginx服务状态
	nginx_list := []interface{}{}
	nginx_msg := []string{}
	nginx_list = append(nginx_list, v.Get("database.nginx"))
	nginx_list = append(nginx_list, v.Get("database.web-service-nginx"))
	nginx_list = append(nginx_list, v.Get("database.kibana"))
	for _, list := range nginx_list {
		for _, nginx := range list.([]interface{}) {
			nginx_map, ok := nginx.(map[string]interface{})
			if !ok {
				continue
			}
			addr := fmt.Sprintf("http://%s:%d", nginx_map["ip"].(string), nginx_map["port"].(int))
			enable, _ := nginx_map["enable"].(bool)
			if !enable {
				continue
			}
			nginx_status, _ := api_get_call_nginx(addr)
			if !nginx_status {
				addr := fmt.Sprintf("https://%s:%d", nginx_map["ip"].(string), nginx_map["port"].(int))
				nginx_status, _ = api_get_call_nginx(addr)
				if !nginx_status {
					nginx_msg = append(nginx_msg, addr)
				}
				// nginx_msg = append(nginx_msg, addr)
			}
		}
	}
	if len(nginx_msg) != 0 {
		send_msg := "# 威胁感知HTTP告警: \n"
		for index, value := range nginx_msg {
			send_msg = send_msg + fmt.Sprintf("%d. 威胁感知HTTP:<font color=\"warning\">%s</font>当前异常! \n", index+1, value)
		}
		send_weixin = send_weixin + send_msg
	}

	if len(results) != 0 || len(app_msg) != 0 || len(kafka_msg) != 0 || len(pg_msg) != 0 || len(zookeeper_msg) != 0 || len(hbase_msg) != 0 || len(nginx_msg) != 0 {
		if production_tag && v.GetBool("database.weixin.enable") {
			send_md(v.GetStringMapString("database.weixin")["webhook"], send_weixin)
		} else {
			fmt.Println(send_weixin)
		}
		if v.GetBool("database.email.enable") {
			mail_port, _ := strconv.Atoi(v.GetStringMapString("database.email")["port"])
			fmt.Println(v.Get("database.email.to_addr"))
			send_email_msg(v.Get("database.email.to_addr").([]interface{}), v.GetStringMapString("database.email")["from_addr"],
				"威胁感知官服邮件告警", send_weixin, v.GetStringMapString("database.email")["smtp"], mail_port,
				v.GetStringMapString("database.email")["login_user"], v.GetStringMapString("database.email")["login_pass"])
		}
	}
}

// 是否是生产环境
var production_tag bool = false

func main() {
	current_dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取脚本当前目录失败")
		return
	}
	config_file_name := "config-test.yaml"
	if production_tag {
		config_file_name = "config.yaml"
	}
	config_file_path := filepath.Join(current_dir, config_file_name)
	v := viper.New()
	v.AddConfigPath(current_dir)
	v.SetConfigFile(config_file_path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Println("配置文件读取错误:", err)
		return
	}
	sleep_time := 10 * time.Second
	if production_tag {
		sleep_time = 5 * time.Minute
	}
	for {
		now := time.Now()
		if now.Hour() >= 8 && now.Hour() < 22 || !production_tag {
			background_task(v, sleep_time)
		} else {
			time.Sleep(10 * time.Minute)
			fmt.Println("不在指定时间范围内，等待一段时间后再次检查")
		}
	}
}
