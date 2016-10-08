package main

import (
	"github.com/jordan9001/rush_n_crush"
	"net/http"
)

func main() {
	go http.ListenAndServe(":8080", http.FileServer(http.Dir("/var/www/site")))
	rush_n_crush.StartServer("/", ":12345", "./run/startup2.cmd")
}
