package main

import (
	"context"
	"os"
	"sync"
	"time"

	mockClient "github.com/2comjie/taoxi-server/cmd/mock/client"
	"github.com/2comjie/wali/core/util"
	waliflag "github.com/2comjie/wali/flag"
	"github.com/2comjie/wali/logx"
	"github.com/2comjie/wali/logx/wlog"
)

var uidNum int
var uidStart uint64
var baseUrl string

var wg sync.WaitGroup

func init() {
	uidNum = waliflag.Int("uid-num", 10)
	uidStart = uint64(waliflag.Int("uid-start", 10000000))
	baseUrl = waliflag.String("base-url", "127.0.0.1:8080")
}

func main() {
	logger := wlog.NewLog(os.Stdout).WithLevel("debug")
	logx.SetLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())
	for index := 0; index < uidNum; index++ {
		uid := uint64(index) + uidStart
		wg.Add(1)
		go func() {
			defer wg.Done()
			runOnePlayer(uid, baseUrl, ctx)
		}()
	}
	util.WaitUntilSignaled()
	cancel()
	wg.Wait()
}

func runOnePlayer(uid uint64, baseUrl string, ctx context.Context) {
	tk := time.NewTicker(time.Millisecond * 100)
	defer tk.Stop()
	mc := mockClient.NewMockClient(uid, baseUrl, ctx)
	lastTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			dt := time.Now().Sub(lastTime)
			mc.Update(dt)
			lastTime = time.Now()
		}
	}
}
