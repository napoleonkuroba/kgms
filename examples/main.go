package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main(){
	path,_:=os.Getwd()
	file, err := os.Open(path+"/resources/计算机网络1计算机网络体系结构.md")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	fmt.Println(string(content))
}
