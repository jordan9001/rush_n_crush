package main

import (
	"github.com/jordan9001/rush_n_crush"
	"net/http"
	"os"
)

func main() {
	go http.ListenAndServe(":8080", http.FileServer(http.Dir("./run/site")))
	var startup_file string
	if len(os.Args) > 1 {
		startup_file = os.Args[1]
	}
	rush_n_crush.StartServer("/", ":12345", startup_file)
}
