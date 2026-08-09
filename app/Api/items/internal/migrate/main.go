package main

import (
	"context"

	"github.com/2comjie/taoxi-server/app/Api/items/internal/store/ent"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	client, err := ent.Open("mysql", "root:123@tcp(127.0.0.1:3306)/taoxi-game?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if err = client.Schema.Create(context.Background()); err != nil {
		panic(err)
	}
}
