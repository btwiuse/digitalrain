package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/wasm/", http.StripPrefix("/wasm/", http.FileServer(http.Dir("wasm"))))
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("Listening on :" + port)
	log.Println("  GopherJS  -> http://localhost:" + port)
	log.Println("  Go WASM   -> http://localhost:" + port + "/wasm/")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
