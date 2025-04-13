package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Handler struct {
	lock   sync.Mutex
	routes map[string]string
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) SetRoutes(routes map[string]string) error {
	// @todo, Check if there is a diff. If so, lock and update.

	h.lock.Lock()
	defer h.lock.Unlock()

	h.routes = routes

	return nil
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	target, ok := h.routes[r.Host]
	if !ok {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	parsed, err := url.Parse(fmt.Sprintf("http://%s:8080", target))
	if err != nil {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	httputil.NewSingleHostReverseProxy(parsed).ServeHTTP(w, r)
}
