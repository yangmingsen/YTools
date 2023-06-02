package main

import (
	"fmt"
	"time"
)

func main() {
	// 创建一个通道用于接收超时信号
	timeout := make(chan bool, 1)

	// 启动一个 goroutine 来执行 IO 流请求
	go func() {
		// 模拟 IO 流请求
		time.Sleep(10 * time.Second)

		// 发送超时信号到通道
		timeout <- true
	}()

	// 使用 select 语句等待超时或完成
	select {
	case <-timeout:
		// 超时
		fmt.Println("请求IO流超时10，自动停止")
		// 执行停止逻辑
		// ...

	case <-time.After(3 * time.Second):
		// 达到超时时间
		fmt.Println("请求IO流超时，自动停止")
		// 执行停止逻辑
		// ...
	}
}
