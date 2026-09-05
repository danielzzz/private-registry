package main

import (
	"log"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/web"
)

func main() {
	addr := ":8080"
	log.Fatal(http.ListenAndServe(addr, web.NewHandler(web.Deps{})))
}
