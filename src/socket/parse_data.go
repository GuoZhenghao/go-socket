package socket

import (
	"common"
	"time"
	"strconv"
	"github.com/jiguorui/crc16"
	"biu"
	"strings"
)

//自定义错误类型
type ArithmeticError struct {
	error //实现error接口
}

//重写Error()方法
func (this *ArithmeticError) Error() string {
	return "自定义的error,error名称为算数不合法"
}

type Data struct {
	header       []byte
	infoType     []byte
	serialNumber []byte
	packageLen   []byte
	deviceId     []byte
	content      []byte
	crc          []byte
	endTag       byte
}

//消息解析
func ParseData(msg []byte) (d *Data, err error) {
	var data Data
	l := len(msg)
	//消息除content部分外，长度为19
	if l < 19 {
		return nil, &ArithmeticError{}
	}
	data.header = msg[0:2]
	data.infoType = msg[2:4]
	data.serialNumber = msg[4:6]
	data.packageLen = msg[6:8]
	data.deviceId = msg[8:16]
	data.content = msg[16 : l-3]
	data.crc = msg[l-3 : l-1]
	data.endTag = msg[l-1]
	return &data, nil
}

//消息转换
func DataToByte(data *Data, cont []byte) []byte {
	var body []byte
	body = append(body, data.header...)
	body = append(body, data.infoType...)
	body = append(body, data.serialNumber...)
	body = append(body, common.Uint16ToByte(uint16(len(cont)+19))...)
	body = append(body, data.deviceId...)
	body = append(body, cont...)
	body = append(body, common.Uint2Byte(crc16.CheckSum(body))...)
	body = append(body, data.endTag)
	return body
}

// 当前时间转 []byte
func ParseTime() []byte{
	timeNow := time.Now().Format("20060102150405")
	content := []byte{}
	yy_o,_ := strconv.ParseUint(timeNow[0:4],10,16)
	yy := uint8(uint16(yy_o) - uint16(1970))
	content = append(content,byte(yy))
	MM,_ := strconv.ParseUint(timeNow[4:6],10,8)
	content = append(content,byte(uint8(MM)))
	dd,_ := strconv.ParseUint(timeNow[6:8],10,8)
	content = append(content,byte(uint8(dd)))
	hh,_ := strconv.ParseUint(timeNow[8:10],10,8)
	content = append(content,byte(uint8(hh)))
	mm,_ := strconv.ParseUint(timeNow[10:12],10,8)
	content = append(content,byte(uint8(mm)))
	ss,_ := strconv.ParseUint(timeNow[12:14],10,8)
    content = append(content,byte(uint8(ss)))
    return content
}

//所有充电口状态解析
func TransChargingPileState(content []byte) map[string]string{
	valueMap := make(map[string]string)
	port := 0
	_date := biu.BytesToBinaryString(content)
	_date = strings.Replace(_date,"[","",-1);
	_date = strings.Replace(_date,"]","",-1);
	_value := strings.Split(_date," ");
	for i := len(_value) - 1;i > -1;i--{
		port++
		valueMap[strconv.Itoa(port)] = tranState(_value[i][4:8])
		port++
		valueMap[strconv.Itoa(port)] = tranState(_value[i][0:4])
	}
	return valueMap
}
//充电桩状态
func tranState(s string) string {
	switch s {
	case "0000":
		s = "空闲"
		break
	case "0001":
		s = "充电中"
		break
	case "0010":
		s = "充满"
		break
	case "0011":
		s = "异常"
		break
	case "0100":
		s = "待连接"
		break
	case "0101":
		s = "未装配"
		break
	case "0110":
		s = "拒充"
		break
	default:
		s = "未知状态"
	}
	return s
}