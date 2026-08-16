package flags

import novaflag "github.com/2comjie/nova/flag"

var ServiceIndex int   // 服务的索引
var ServiceName string // 服务名称
var Env string         // 当前环境 Dev Local Prod
const (
	Dev   = "Dev"
	Local = "Local"
	Prod  = "Prod"
)

func init() {
	ServiceIndex = novaflag.Int("service-index", -1)
	ServiceName = novaflag.String("service-name", "")
	Env = novaflag.String("env", Local)
	if ServiceIndex == -1 {
		panic("service-index is required")
	}
	if ServiceName == "" {
		panic("service-name is required")
	}
}
