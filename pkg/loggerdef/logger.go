package loggerdef

import "github.com/2comjie/nova/logx/logdef"

type Log struct {
	Path   string
	Level  logdef.Level
	Stdout bool
}
