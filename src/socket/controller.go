package socket

import (
	"configure"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
	"xslog"
)

/*
	指令发送逻辑:
		1. 监听配置文件指定的端口,根据传入的URL判断执行的函数. 函数名,imei和msg都必须传,如果不需要则传空
			- 示例: /FunName?imei=12345678&msg=
		2. 在 socket/chargeingPile.go 下添加 服务器发送到充电桩 的函数, 需要满足以下要求，否则不会处理该命令的返回
			- 函数命名需要与enum中的指令相同(首字母必须大写)
			- 必须是 ChargeingPile 的扩展方法
			- 必须将发送的指令写入 globalToPileMap 中, key值统一为 imei+"-AppDownload-"+serialNumber
		3. 在 socket/socket_service.go/InfoTypeVerify() 添加 对充电桩回应信息 的处理,并将校验结果写回到 globalFromPileMap 中. key值如上述格式
			- 可以在 globalToPileMap 读取该命令的发送格式以作校验
*/

type resultMap struct {
	Code int16
	Body map[string]string
}

var rwMutex *sync.RWMutex

// 从充电桩返回的信息的 key/value 对
var globalFromPileMap = make(map[string]resultMap, 1)

// 服务器发送的信息的 key/value 对
var globalToPileMap = make(map[string][]byte, 1)

// 启动服务监听(FROM 前端/对外服务端)
func (d *Devices) StartSocketClient(ipaddr string, port string, network string) {
	defer func() {
		if err := recover(); err != nil {
			xslog.Showmsg(err)
		}
	}()

	var chargeingPile ChargeingPile
	rwMutex = new(sync.RWMutex)
	// 为所有方法添加handle
	// 通过typeof中的method方法，得到chargeingPile中的所有方法名
	t := reflect.TypeOf(&chargeingPile)
	for i := 0; i < t.NumMethod(); i++ {
		http.HandleFunc("/"+t.Method(i).Name, getStatus)
	}
	listenPort := configure.ReadConfigByKey("./init.ini", "Net", "listenPort")
	err := http.ListenAndServe(":"+listenPort, nil)
	if err != nil {
		xslog.Log("", err)
	}
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	result := resultMap{
		Code: 0,
		Body: nil,
	}
	defer func() {
		if err := recover(); err != nil {
			xslog.Showmsg(err)
		}
		//处理完成后，返回结果信息
		jsonResult, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", jsonResult)
	}()
	r.ParseForm()
	imei := r.Form["imei"][0]
	//判断是否登陆
	conn, ok := globalDevices.device[imei]
	if ok {
		// 设置参数
		var chargeingPile ChargeingPile
		v := reflect.ValueOf(&chargeingPile)
		cmd := strings.Replace(r.URL.Path, "/", "", -1)
		args := []reflect.Value{reflect.ValueOf(conn)}
		msg := r.Form["msg"]
		args = append(args, reflect.ValueOf(string(imei)), reflect.ValueOf(msg[0]))
		// 执行反射方法
		xslog.Debug1("开始执行反射方法: " + cmd)
		values := v.MethodByName(cmd).Call(args)
		xslog.Debug1("反射方法执行完毕: " + cmd)
		// 得到执行后的结果
		hashId := values[0].String()
		//不断循环等待充电桩返回的结果
		for i := 0; i < 100; i++ {
			//判断是否与发送信息匹配
			valueOfRequestMap, ok := globalFromPileMap[hashId]
			if ok {
				result = valueOfRequestMap
				// 删除传入充电桩和返回的信息
				delete(globalToPileMap, hashId)
				delete(globalFromPileMap, hashId)
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
	//程序执行完成后会执行上方的defer，用于返回结果
}
