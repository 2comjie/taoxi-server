package router

import (
	"path/filepath"
	"testing"

	"github.com/2comjie/nova/app/gate"
	"github.com/2comjie/nova/config"
	"github.com/2comjie/nova/config/file"
)

func TestRouteConfig(t *testing.T) {
	for _, env := range []string{"Local", "Dev"} {
		t.Run(env, func(t *testing.T) {
			center := config.New(config.WithSource(file.NewSource(filepath.Join("../../../configs", env))))
			if err := center.Load(); err != nil {
				t.Fatal(err)
			}
			defer center.Close()

			routes, err := config.Get[[]gate.Route](center, "gate.routes")
			if err != nil {
				t.Fatal(err)
			}
			root := Init()
			if err = root.Add(routes...); err != nil {
				t.Fatal(err)
			}
			if err = root.Freeze(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
