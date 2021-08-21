package configs

import (
	"encoding/json"
	"os"
)

//服务端配置
type AppConfig struct {
	AppName    string `json:"app_name"`
	Port       string `json:"port"`
	Mode       string `json:"mode"`
}

//数据库配置
type SqlConfig struct {
	Driver     string `json:"driver"`
	SQLName    string `json:"sql_name"`
	SQLPasswd  string `json:"sql_passwd"`
	DataBase   string `json:"data_base"`
	SQLHost    string `json:"sql_host"`
	SQLPort    string `json:"sql_port"`
	CasbinConf string `json:"casbin_conf"`
}

//初始化服务器配置
func InitConfig() *AppConfig {
	str, _ := os.Getwd()
	file, err := os.Open(str + "/configs/config.json")
	if err != nil {
		panic(err.Error())
	}
	decoder := json.NewDecoder(file)
	conf := AppConfig{}
	err = decoder.Decode(&conf)
	if err != nil {
		panic(err.Error())
	}
	return &conf
}

//初始化数据库服务
func InitSQLConfig() *SqlConfig {
	str, _ := os.Getwd()
	file, err := os.Open(str + "/configs/config.json")
	if err != nil {
		panic(err.Error())
	}
	decoder := json.NewDecoder(file)
	conf := SqlConfig{}
	err = decoder.Decode(&conf)
	if err != nil {
		panic(err.Error())
	}
	return &conf
}
