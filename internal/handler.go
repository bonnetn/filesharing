package internal

import (
	"errors"
	"net/http"
	"strings"
)

const (
	routeAPI = "/api/"
)

func NewHandler(get FileshareGetter, create FileshareCreator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(routeAPI, func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(get, create, w, r)
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

func serveHTTP(get FileshareGetter, create FileshareCreator, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	resourceName := strings.TrimPrefix(path, routeAPI)

	var err error
	switch r.Method {
	case http.MethodGet:
		err = get.Get(r.Context(), w, resourceName)

	case http.MethodPost:
		err = create.Create(r.Context(), w, resourceName, r)

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
	case errors.Is(err, &BadRequestError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, &NotFoundError{}):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
