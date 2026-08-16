package instruction

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/2comjie/nova/config"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	"github.com/2comjie/nova/logx/wlog"
	"github.com/2comjie/taoxi-server/flags"
	"github.com/2comjie/taoxi-server/pkg/loggerdef"
)

func InitLogger(center config.Config) error {
	logConfig := &loggerdef.Log{}

	value := center.Value("bootstrap.logger")
	if value == nil {
		return fmt.Errorf("nodeDeploy: 缺少 logger 配置")
	}
	if err := value.Scan(logConfig); err != nil {
		return fmt.Errorf("nodeDeploy: 解析 logger 配置失败: %w", err)
	}
	writers := make([]io.Writer, 0, 2)
	if logConfig.Path != "" {
		if err := os.MkdirAll(logConfig.Path, 0o755); err != nil {
			return fmt.Errorf(
				"nodeDeploy: 创建日志目录失败 path=%s: %w",
				logConfig.Path,
				err,
			)
		}

		logPath := filepath.Join(
			logConfig.Path,
			fmt.Sprintf(
				"%s_%d_info.log",
				flags.ServiceName,
				flags.ServiceIndex,
			),
		)

		logFile, err := os.OpenFile(
			logPath,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0o644,
		)
		if err != nil {
			return fmt.Errorf(
				"nodeDeploy: 打开日志文件失败 path=%s: %w",
				logPath,
				err,
			)
		}

		writers = append(writers, logFile)
	}

	if logConfig.Stdout {
		writers = append(writers, os.Stdout)
	}

	if len(writers) == 0 {
		return fmt.Errorf("nodeDeploy: 日志文件和 stdout 不能同时关闭")
	}
	logger := wlog.NewLog(io.MultiWriter(writers...)).WithLevel(logConfig.Level).WithFields(logdef.Fields{"service": flags.ServiceName, "service_index": flags.ServiceIndex})
	logx.SetLogger(logger)
	logx.Infof("nodeDeploy: 日志初始化完成 level=%s stdout=%t", logConfig.Level, logConfig.Stdout)
	return nil
}
