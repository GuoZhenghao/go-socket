package xslog
import (
	"fmt"
	"time"
)
const timeLayout = "2006-01-02 03:04:05"


func Log(msg string,err error){
	if err!=nil{
		fmt.Printf("%s: %s; err: %s \n",time.Now().Format(timeLayout),msg,err.Error())
		panic(err)
	}
}

func Showmsg(msg interface{}){
	fmt.Printf("%s: %v \n",time.Now().Format(timeLayout),msg)
}

func MakeError(msg interface{}) error{
	return fmt.Errorf("%s: %v",time.Now().Format(timeLayout), msg)
}
func Debug1(msg interface{}){
	fmt.Printf("%s: %v \n",time.Now().Format(timeLayout),msg)
}