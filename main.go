package main

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
	"io/ioutil"
	"knowledge-graph-management-system/configs"
	"knowledge-graph-management-system/controllers"
	"knowledge-graph-management-system/models"
	"log"
	"os"
	"strings"
	"time"
)

var Mysql *xorm.Engine
var Cache *models.Cache

func main() {
	config := configs.InitConfig()
	Mysql = configs.NewMySQLEngine()
	if Mysql == nil {
		log.Panic("mysql starts error")
	}

	app := iris.New()
	app.Use(Cors)
	//region 启用session
	sessManager := sessions.New(sessions.Config{
		Cookie:  "sessioncookie",
		Expires: time.Hour,
	})
	//endregion

	Sync()

	//region 注册路由
	app.HandleDir("/resource", "./static")

	dataParty := app.Party("/") //注册信息管理控制器路由
	data := mvc.New(dataParty)
	data.Register(sessManager.Start, Mysql, Cache)
	data.Handle(new(controllers.Controller))
	//endregion

	app.Configure(iris.WithConfiguration(iris.Configuration{
		Charset: "UTF-8",
	}))
	err := app.Run(
		iris.Addr(":"+config.Port),
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithOptimizations,
	)
	if err != nil {
		log.Panic(err.Error())
	}
}

func Cors(ctx iris.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Next()
}

func Sync() {
	files := make([]models.Files, 0)
	cache := make(map[string][]string)
	err := Mysql.Find(&files)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, fileInfo := range files {
		path, _ := os.Getwd()
		path = path + "/resources/" + fileInfo.Name + ".md"
		file, err := os.Open(path)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return
		}
		lines := strings.Split(string(content), "\n")
		cache[fileInfo.Name] = lines
		file.Close()
	}
	cacheData := &models.Cache{
		FileContent: cache,
		Files:       files,
	}
	Cache = cacheData
}
