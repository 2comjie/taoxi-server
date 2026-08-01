package external

import (
	"fmt"
	"strings"

	entSQL "entgo.io/ent/dialect/sql"
	"github.com/2comjie/wali/config"
	_ "github.com/go-sql-driver/mysql"
)

type MysqlConfig struct {
	DSN      string `json:"dsn"`
	OpenConn int    `json:"open_conn"`
	IdleConn int    `json:"idle_conn"`
}

var (
	entDrivers = make(map[string]*entSQL.Driver)
)

func InitMysql(center config.Config) error {
	entDrivers = make(map[string]*entSQL.Driver)
	configs := map[string]*MysqlConfig{}
	mysqlConfig := center.Value("bootstrap.mysql")
	if mysqlConfig == nil {
		return fmt.Errorf("nodeDeploy: 缺少 mysql 配置")
	}
	err := mysqlConfig.Scan(&configs)
	if err != nil {
		return fmt.Errorf("nodeDeploy: 解析 mysql 配置失败: %w", err)
	}
	for name, options := range configs {
		driver, err := openEntDriver(name, *options)
		if err != nil {
			return err
		}
		entDrivers[name] = driver
	}
	return nil
}

func openEntDriver(key string, cfg MysqlConfig) (*entSQL.Driver, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("mysql %q dsn is required", key)
	}
	drv, err := entSQL.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql %q: %w", key, err)
	}

	db := drv.DB()
	if cfg.OpenConn <= 0 {
		cfg.OpenConn = 200
	}
	if cfg.IdleConn <= 0 {
		cfg.IdleConn = 200
	}
	db.SetMaxOpenConns(cfg.OpenConn)
	db.SetMaxIdleConns(cfg.IdleConn)

	return drv, nil
}

func GetEntDriver(name string) *entSQL.Driver {
	return entDrivers[name]
}

func MysqlGame() *entSQL.Driver {
	return GetEntDriver("game")
}
func MysqlUser() *entSQL.Driver {
	return GetEntDriver("user")
}
