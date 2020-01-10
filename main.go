package main

import (
	"github.com/gorilla/mux"
	"github.com/raphaelreyna/latte/server"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/generate", server.HandleGenerate)
	log.Fatal(http.ListenAndServe(":5000", r))
}
