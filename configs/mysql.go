package configs

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"knowledge-graph-management-system/models"
	"log"
	"xorm.io/core"
)

func NewMySQLEngine() *xorm.Engine {
	conf := InitSQLConfig()
	engine, err := xorm.NewEngine(conf.Driver, conf.SQLName+":"+conf.SQLPasswd+"@tcp("+conf.SQLHost+":3306)"+"/"+conf.DataBase+"?charset=utf8")
	if err != nil {
		log.Panic(err.Error())
	}
	engine.SetTableMapper(core.SameMapper{})
	engine.SetColumnMapper(core.SameMapper{})
	err = engine.Sync2(
		new(models.Files),
		new(models.KeyIndex),
		new(models.Subject),
	)

	if err != nil {
		log.Panic(err.Error())
	}
	engine.ShowSQL(false)
	engine.SetMaxOpenConns(10)
	engine.SetMaxIdleConns(5)
	return engine
}
