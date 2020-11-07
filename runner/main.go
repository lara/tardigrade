package main

import (
	"runner/websocket"

	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/echo", websocket.Start)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
