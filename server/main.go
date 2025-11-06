package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})
	fmt.Println("Hello World")
	fmt.Println("Server started on port 8081")
	http.ListenAndServe(":8081", nil)
}
