package handler

import (
	"errors"
	"github.com/bonnetn/filesharing/endpoint"
	"net/http"
	"strings"
)

const (
	routeAPI       = "/api/"
)

func NewHandler(c *endpoint.ConnectionController) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(routeAPI, &apiHandler{
		Controller: c,
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		http.ServeFile(w, req, "index.html")
	})

	return mux
}

type apiHandler struct {
	Controller *endpoint.ConnectionController
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	resourceName := strings.TrimPrefix(path, routeAPI)


	var err error
	switch r.Method {
	case http.MethodGet:
		err = h.Controller.Get(r.Context(), w, resourceName)

	case http.MethodPost:
		err = h.Controller.Post(r.Context(), w, resourceName, r)

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	writeErrors(w, err)
}


func writeErrors(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, &endpoint.BadRequestError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, &endpoint.NotFoundError{}):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}