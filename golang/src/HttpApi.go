package main

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/driver/mysql"
	"io"
	"log"
	"net/http"
)

var Db *gorm.DB

//Ret ...
type Ret struct {
	Code int    `json:"code,int"`
	Data string `json:"data"`
}

func printRequest(w http.ResponseWriter, r *http.Request, ret *Ret) {

	fmt.Println("r.Form=", r.Form) //这些信息是输出到服务器端的打印信息 , Get参数

	//ret.Code = 200
	//ret.Data = "提交成功"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	retJSON, _ := json.Marshal(ret)
	io.WriteString(w, string(retJSON))
}

func sayMore(w http.ResponseWriter, r *http.Request) {
	ret := new(Ret)
	r.ParseForm() //解析参数，默认是不会解析的
	ret.Code, ret.Data = handleDormRequ(r.Form)
	printRequest(w, r, ret)
}

//连接数据库
func connectToDB() {
	dsn := "web2020:123456@tcp(101.133.163.0:3306)/web2020?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("------------连接数据库成功-----------")
}

func test(w http.ResponseWriter, r *http.Request)  {
	fmt.Fprint(w, "hello")
}

func main() {
	connectToDB()
	http.HandleFunc("/api/room/order", sayMore) //设置访问的路径
	http.HandleFunc("/test", test)
	err := http.ListenAndServe(":8080", nil)    //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
