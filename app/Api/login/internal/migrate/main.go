package main

import (
	"context"

	"github.com/2comjie/taoxi-server/app/Api/login/internal/store/ent"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	client, err := ent.Open("mysql", "root:123@tcp(127.0.0.1:3306)/taoxi-user?charset=utf8mb4")
	if err != nil {
		panic(err)
	}
	err = client.Schema.Create(context.Background())
	if err != nil {
		panic(err)
	}
}
