package main

import (
	"configure"
	"socket"
)

func main() {
	//初始化，读取配置文件
	ip := configure.ReadConfigByKey("./init.ini", "Net", "serviceIp")
	port := configure.ReadConfigByKey("./init.ini", "Net", "servicePort")
	network := configure.ReadConfigByKey("./init.ini", "Net", "network")
	socket.Start(ip, port, network)
}
