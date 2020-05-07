package handlers

import (
	"net/http"
)

// RenderHome Rendering the Home Page
func RenderHome(response http.ResponseWriter, request *http.Request) {
	http.ServeFile(response, request, "views/index.html")
}
