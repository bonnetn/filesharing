package internal

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

const (
	routeAPI = "/"
)

func NewHandler(get FileshareGetter, create FileshareCreator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	mux.HandleFunc(routeAPI, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		serveHTTP(get, create, w, r)
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

	writeErrors(w, r, err)
}

func writeErrors(w http.ResponseWriter, req *http.Request, err error) {
	if err == nil {
		return
	}

	log.Printf("error: %v", err)
	switch {
	case errors.Is(err, &LogOnlyError{}):
		return // Don't do anything.

	case errors.Is(err, &BadRequestError{}):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, &NotFoundError{}):
		http.ServeFile(w, req, "index.html") // If the resource is not found, redirect the user to the form.

	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
