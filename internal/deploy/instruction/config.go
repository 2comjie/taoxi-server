package instruction

import (
	"fmt"
	"path/filepath"

	"github.com/2comjie/nova/config"
	"github.com/2comjie/nova/config/file"
	"github.com/2comjie/nova/core/util"
	"github.com/2comjie/taoxi-server/flags"
	"github.com/2comjie/taoxi-server/pkg/loggerdef"
)

type Config struct {
	Logger *loggerdef.Log `json:"logger"`
}

func InitConfig() (config.Config, error) {
	var envDir string
	switch flags.Env {
	case flags.Local:
		envDir = "Local"
	case flags.Dev:
		envDir = "Dev"
	case flags.Prod:
		envDir = "Prod"
	default:
		return nil, fmt.Errorf("nodeDeploy: 不支持的运行环境 %q", flags.Env)
	}
	path, err := util.GetModuleRootDir()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, "configs", envDir)
	source := file.NewSource(path)
	center := config.New(config.WithSource(source))
	return center, nil
}
