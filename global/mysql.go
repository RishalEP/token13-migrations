package global

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	rawLog "log"
	"os"
	"time"
)

type MysqlDBPool struct {
	dbs map[string]*gorm.DB
}

func NewMysqlDBPool() *MysqlDBPool {
	return &MysqlDBPool{
		dbs: make(map[string]*gorm.DB),
	}
}

func (p MysqlDBPool) AddDB(name string, sqlInfo string, maxConn int, idleConn int, maxLeftTime time.Duration) error {
	mysqlConfig := mysql.Config{
		//DSN: "root:@tcp(127.0.0.1:3306)/token13_app?parseTime=True",

		DSN:                       GConfig.Viper.GetString("db-data-source"),
		DefaultStringSize:         191,   // string 类型字段的默认长度
		SkipInitializeWithVersion: false, // 根据版本自动配置

	}
	logC := logger.Config{
		SlowThreshold: time.Second,
		LogLevel:      logger.Silent, // 关闭SQL日志，如需打开设置为logger.Info, 关闭设置为logger.Silent (logger.LogLevel(0))
		Colorful:      true,
	}

	logConfig := logger.New(rawLog.New(os.Stdout, "\r\n", rawLog.LstdFlags|rawLog.Lshortfile), logC)
	db, err := gorm.Open(mysql.New(mysqlConfig), &gorm.Config{
		Logger: logConfig,
	})
	if err != nil {
		return err
	}
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		return err
	}
	sqlDb, _ := db.DB()
	sqlDb.SetMaxOpenConns(maxConn)
	sqlDb.SetMaxIdleConns(idleConn)
	sqlDb.SetConnMaxLifetime(maxLeftTime)
	if err = sqlDb.Ping(); err != nil {
		return err
	}
	p.dbs[name] = db

	return nil
}

func (p MysqlDBPool) GetDB(name string) *gorm.DB {
	if db, ok := p.dbs[name]; ok {
		return db
	}

	return nil
}

func (p MysqlDBPool) ReleasePool() {
	for _, db := range p.dbs {
		sqlDb, _ := db.DB()
		_ = sqlDb.Close()
	}
}

var defaultPool *MysqlDBPool

func init() {
	defaultPool = NewMysqlDBPool()
}

func AddDB(name string, sqlInfo string, maxConn int, idleConn int, maxLeftTime time.Duration) error {
	return defaultPool.AddDB(name, sqlInfo, maxConn, idleConn, maxLeftTime)
}

func GetDB(name string) *gorm.DB {
	return defaultPool.GetDB(name)
}

func MustGetDB(name string) *gorm.DB {
	db := GetDB(name)
	if db == nil {
		panic("DB " + name + " not exist")
	}
	return db
}

func ReleaseMysqlDBPool() {
	defaultPool.ReleasePool()
}
