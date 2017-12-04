package socket

import (
	"common"
	"fmt"
	"net"
	"strconv"
	"strings"
	"xslog"

	"github.com/jiguorui/crc16"
)

type ChargeingPile struct {
	PileId string
	// 远程socket服务器IP
	RegSocketIP   string
	RegSocketPort uint16
	// 本地socket客户端IP
	BizSocketIP   string
	BizSocketPort uint16
	// 分隔符
	Separator string
	// 包头包尾表示
	PackageHead uint16
	PackageTail uint8
	// 版本号, 4byte
	Version string
}

const PackageHead = 26729
const PackageTail = 22

var AppDownloadSerialNo = uint16(1)
var ChargingPortSwitchSerialNo = uint16(1)
var GetChargingPortSerialNo = uint16(1)

// ---------------------业务服务  BEGIN-----------------------------
// 所有的发送服务需要统一的输入参数(通过反射调用的方法,默认设置了两个参数(conn,serialNo)), 如果有特例, 需要在主函数添加
// 必须 返回Key, 作为从 globalRequestMsgMap 中读取相应的返回值
// 必须 传入 conn(链接), imei(唯一标示), msg(要处理的信息)(无处理信息则传空)
//************0106设置app下载地址的方法注释写的比较全可以参考************

// 设置APP下载地址(0106)
func (c *ChargeingPile) AppDownload(conn *net.TCPConn, imei string, msg string) string {
	var data Data
	//得到这条设置App下载链接的信息序列号
	serilaNo := getSerialNo(&AppDownloadSerialNo)
	//拼接报文信息
	data.infoType = []byte{byte(0x01), byte(0x06)}
	data.serialNumber = common.Uint2Byte(uint16(serilaNo))
	//19为除去信息位的报文长度
	data.packageLen = common.Uint16ToByte(uint16(len(msg) + 19))
	data.deviceId = transIMEI(imei)
	data.content = []byte(msg)
	body := data2Byte(data)
	// 充电桩回应时，会将序列号加1，为了方便后面与收到的消息做验证，存入时直接将序列号+1
	id_key := fmt.Sprintf("%s-AppDownload-%d", imei, common.Uint2Byte(uint16(serilaNo+1)))
	xslog.Debug1("Send msg to pile id_key: " + id_key)
	//将这条报文存入到发送集合中
	globalToPileMap[id_key] = body
	//向充电桩发送报文
	conn.Write(body)
	return id_key
}

// 控制各充电口开关(0404)
//******约定******
//传入值样式：充电口，状态    如：1,1
func (c *ChargeingPile) ChargingPortSwitch(conn *net.TCPConn, imei string, msg string) string {
	var data Data
	//处理传入的msg，转换收到的充电口序号和状态号
	_msg := strings.Split(msg, ",")
	if len(_msg) == 2 {
		chargingPort, err := strconv.ParseUint(_msg[0], 10, 8)
		if err != nil {
			panic(err.Error())
		}
		chargingState, err := strconv.ParseUint(_msg[1], 10, 8)
		if err != nil {
			panic(err.Error())
		}
		data.content = []byte{uint8(chargingPort), uint8(chargingState)}
	}
	serilaNo := getSerialNo(&ChargingPortSwitchSerialNo)
	data.infoType = []byte{byte(0x04), byte(0x04)}
	data.serialNumber = common.Uint2Byte(uint16(serilaNo))
	data.packageLen = common.Uint16ToByte(21)
	data.deviceId = transIMEI(imei)
	body := data2Byte(data)
	id_key := fmt.Sprintf("%s-ChargingPortSwitch-%d", imei, common.Uint2Byte(uint16(serilaNo+1)))
	xslog.Debug1("Send msg to pile id_key: " + id_key)
	globalToPileMap[id_key] = body
	conn.Write(body)
	return id_key
}

// 获取所有充电口状态(0605)
func (c *ChargeingPile) GetChargingPort(conn *net.TCPConn, imei string, msg string) string {
	var data Data
	serilaNo := getSerialNo(&GetChargingPortSerialNo)
	data.infoType = []byte{byte(0x06), byte(0x05)}
	data.serialNumber = common.Uint2Byte(uint16(serilaNo))
	data.packageLen = common.Uint16ToByte(19)
	data.content = []byte{}
	data.deviceId = transIMEI(imei)
	body := data2Byte(data)
	//回复的报文中，破充电桩序列号瞎变不知道规律，置1
	//id_key := fmt.Sprintf("%s-GetChargingPort-%d", imei, common.Uint2Byte(uint16(serilaNo + 17)))
	id_key := fmt.Sprintf("%s-GetChargingPort-%d", imei, common.Uint2Byte(uint16(1)))
	xslog.Debug1("Send msg to pile id_key: " + id_key)
	globalToPileMap[id_key] = body
	conn.Write(body)
	return id_key
}

func getSerialNo(funcNameSerialNo *uint16) uint16 {
	//加入写锁 防止同时请求时发生错误
	defer rwMutex.Unlock()
	rwMutex.Lock()
	if *funcNameSerialNo > 65500{
		*funcNameSerialNo = 1
	}
	serilaNo := *funcNameSerialNo
	*funcNameSerialNo = *funcNameSerialNo + 2

	return serilaNo
}

func transIMEI(imei string) []byte {
	result := make([]byte, 8)
	for i := 0; i < 16; i = i + 2 {
		temp, _ := strconv.ParseUint(imei[i:i+2], 16, 8)
		result[i/2] = uint8(temp)
	}
	return result
}

func data2Byte(data Data) []byte {
	var body []byte
	data.header = common.Uint2Byte(PackageHead)
	data.endTag = byte(PackageTail)
	body = append(body, data.header...)
	body = append(body, data.infoType...)
	body = append(body, data.serialNumber...)
	body = append(body, data.packageLen...)
	body = append(body, data.deviceId...)
	body = append(body, data.content...)
	// crc 校验单独进行
	body = append(body, common.Uint2Byte(crc16.CheckSum(body))...)
	body = append(body, data.endTag)
	return body
}
