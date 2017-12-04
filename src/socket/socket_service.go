package socket

import (
	"bufio"
	"bytes"
	"common"
	"enum"
	"fmt"
	"log"
	"net"
	_ "strconv"
	"xslog"
)
//存储所有登陆的设备
var globalDevices *Devices = &Devices{make(map[string]net.Conn)}

type Devices struct {
	device map[string]net.Conn
}
//移除离线设备
func (d *Devices) Leave(ipaddr string) {
	conn, ok := d.device[ipaddr]
	if !ok {
		fmt.Printf("%v设备不存在\n", ipaddr)
	}
	conn.Close()             //如果存在用户就断开链接。
	delete(d.device, ipaddr) //将用户从字典中删除。
	fmt.Printf("%s设备断开\n", ipaddr)
}
//信息类别验证
func InfoTypeVerify(data *Data, line []byte, conn net.Conn) {
	//得到信息类型，用于后面做对应的操作
	infotype := fmt.Sprintf("%x", data.infoType)
	//回应报文中的信息部分
	var content []byte
	switch infotype {
	//登陆0103
	case enum.Login:
		//获取该设备的imei，并添加到登陆的设备列表中
		imei := fmt.Sprintf("%x", data.deviceId)
		globalDevices.device[imei] = conn
		//拼接回应报文
		content = append(content, []byte{0, 4}...)
		content = append(content, common.ParseIpAddr(conn.RemoteAddr().String())...)
		content = append(content, []byte{0, 4}...)
		content = append(content, common.ParseIpAddr("223.223.200.50:9523")...)
		content = append(content, []byte{0, 60}...)
		//服务器回应登陆报文
		SendToObject(DataToByte(data, content), conn)
		break
	//心跳检测0801
	case enum.HeartbeatDetection:
		//回应部分信息内容为空
		SendToObject(DataToByte(data, content), conn)
		break
	//App下载地址0106
	case enum.AppDownload:
		//App下载地址由前端调用，电桩回应报文需要与发送来的地址做匹配验证，故按照如下方式，确保key值唯一的情况下保存
		id_key := fmt.Sprintf("%s-AppDownload-%d", fmt.Sprintf("%x", data.deviceId), data.serialNumber)
		xslog.Debug1("APPDownLoad处理,id_key:" + id_key)
		//得到服务器主动发送时的报文
		toPileMsgByte, ok := globalToPileMap[id_key]
		if ok {
			toPileMsgData, _ := ParseData(toPileMsgByte)
			fmt.Printf("toPileMsgByte: %v,\nFromPileMsgByte: %v", toPileMsgByte, line)
			//判断发送的和接收的报文是否一致，判断是否设置成功
			if bytes.Compare(toPileMsgData.content, data.content) == 0 {
				result := resultMap{
					Code: 1,
					Body: nil,
				}
				globalFromPileMap[id_key] = result
				break
			}
		}
		result := resultMap{
			Code: 0,
			Body: nil,
		}
		globalFromPileMap[id_key] = result
		break
	case enum.AssemblyInformation:
		SendToObject(DataToByte(data, content), conn)
		break
	case enum.RecoveryTime:
		content = append(content, ParseTime()...)
		SendToObject(DataToByte(data, content), conn)
	case enum.ChargingPortSwitch:
		id_key := fmt.Sprintf("%s-ChargingPortSwitch-%d", fmt.Sprintf("%x", data.deviceId), data.serialNumber)
		xslog.Debug1("ChargingPortSwitch处理,id_key:" + id_key)
		toPileMsgByte, ok := globalToPileMap[id_key]
		if ok {
			fmt.Printf("toPileMsgByte: %v,\nFromPileMsgByte: %v", toPileMsgByte, line)
			toPileMsgData, _ := ParseData(toPileMsgByte)
			if toPileMsgData.content[0] == uint8(1) {
				result := resultMap{
					Code: 1,
					Body: nil,
				}
				globalFromPileMap[id_key] = result
				break
			}
		}
		result := resultMap{
			Code: 0,
			Body: nil,
		}
		globalFromPileMap[id_key] = result
		break
	case enum.GetChargingPort:
		//		id_key := fmt.Sprintf("%s-GetChargingPort-%d", fmt.Sprintf("%x", data.deviceId), data.serialNumber)
		id_key := fmt.Sprintf("%s-GetChargingPort-%d", fmt.Sprintf("%x", data.deviceId), common.Uint2Byte(uint16(1)))
		xslog.Debug1("GetChargingPort处理,id_key:" + id_key)
		toPileMsgByte, ok := globalToPileMap[id_key]
		if ok {
			fmt.Printf("toPileMsgByte: %v,\nFromPileMsgByte: %v", toPileMsgByte, line)
			_date, err := ParseData(line)
			if err == nil {
				_content := _date.content
				_body := TransChargingPileState(_content)
				result := resultMap{
					Code: 1,
					Body: _body,
				}
				globalFromPileMap[id_key] = result
				break
			}
		}
		result := resultMap{
			Code: 0,
			Body: nil,
		}
		globalFromPileMap[id_key] = result
		break

	defult:
		result := resultMap{
			Code: 0,
			Body: nil,
		}
		break
	}
}

//服务器被动回应消息
func SendToObject(msg []byte, conn net.Conn) {
	fmt.Println("服务器被动回应充电桩消息:", msg)
	conn.Write(msg)
}

//服务器主动发送消息
func (d *Devices) Send(msg []byte, ipaddr string) {
	conn := d.device[ipaddr]
	conn.Write(msg)
	fmt.Println("服务器主动发送消息")
}

//处理收到的报文
func (d *Devices) Handle_Conn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		//该项目的报文规定包结束标示是16H
		line, err := r.ReadBytes(byte(0x16))
		fmt.Println("收到消息:", line)
		if err != nil {
			break
		}
		//将收到的报文解析，转换为对象
		data, err := ParseData(line)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println(line)
			break
		}
		go InfoTypeVerify(data, line, conn)
	}
}

//启动socket服务器
func (d *Devices) StartSocketService(ipaddr string, port string, network string) {
	addr := fmt.Sprintf("%s:%s", ipaddr, port)
	listener, err := net.Listen(network, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	//不断监听
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		//处理收到的报文
		go globalDevices.Handle_Conn(conn)
	}
}

func Start(ipaddr string, port string, network string) {
	//socket服务
	go globalDevices.StartSocketService(ipaddr, port, network)
	//监听
	globalDevices.StartSocketClient(ipaddr, port, network)
}
