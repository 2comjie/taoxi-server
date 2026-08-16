package pprof

import (
	"context"
	"errors"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"time"

	"github.com/2comjie/nova/app"
	"github.com/2comjie/nova/core/help"
	"github.com/2comjie/nova/logx"
	"github.com/spf13/cast"
)

func StartPprof(svc string, port int) app.Component {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", httppprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", httppprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)

	server := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", cast.ToString(port)),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &app.CommonComponent{
		MName: svc + "-pprof",
		MStart: func() error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}
			baseURL := "http://" + listener.Addr().String() + "/debug/pprof"
			help.SafeGo(func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logx.Errorf("pprof: 服务异常退出 service=%s err=%v", svc, err)
				}
			})
			logx.Infof("pprof: 服务启动 service=%s url=%s/", svc, baseURL)
			logx.Infof("pprof: Heap 分析命令: go tool pprof -http=:0 %s/heap", baseURL)
			logx.Infof("pprof: CPU 分析命令: go tool pprof -http=:0 '%s/profile?seconds=30'", baseURL)
			return nil
		},
		MShutdown: func(ctx context.Context) error {
			if err := server.Shutdown(ctx); err != nil {
				return errors.Join(err, server.Close())
			}
			return nil
		},
	}
}
