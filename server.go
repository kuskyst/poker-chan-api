package main

import (
	"log"
	"net/http"
	"poker-chan-api/controllers"
)

func main() {

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		controllers.ServeWs(w, r)
	})

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
