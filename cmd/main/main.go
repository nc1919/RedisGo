package main

import (
	"RedisGo/internal/parser"
	"log"
)

func main() {
	srv := parser.NewServer(":6380")
	log.Fatal(srv.ListenAndServe())
}
