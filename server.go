package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {

	log.Println("Server will start at http://localhost:8000/")

	route := mux.NewRouter()

	AddApproutes(route)

	log.Fatal(http.ListenAndServe(":8000", route))
}
