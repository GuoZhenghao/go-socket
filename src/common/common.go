package common

import (
	"bytes"
	"encoding/binary"
	"log"
	"strconv"
	"strings"
)

func Uint2Byte(body uint16) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, body)
	if err != nil {
		log.Println(err)
	}
	return buf.Bytes()
}

func Uint16ToByte(par uint16) []byte {
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, par)
	return bs
}

func Uint32ToByte(par uint32) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, par)
	return bs
}

//解析IP
func ParseIpAddr(a string) []byte {
	var ipaddr []byte
	addr := strings.Split(a, ":")

	bits := strings.Split(addr[0], ".")
	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])
	var sum int64
	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)
	ip := Uint32ToByte(uint32(sum))
	ipaddr = append(ipaddr, ip...)

	p, _ := strconv.Atoi(addr[1])
	port := Uint16ToByte(uint16(p))
	ipaddr = append(ipaddr, port...)
	return ipaddr
}
