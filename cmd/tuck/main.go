package main

import (
	"log"

	"liuyu.xin/tuck/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
