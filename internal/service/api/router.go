package api

import (
	"log/slog"
	"net/http"
)

// Middleware represents a HTTP middleware function
type Middleware func(http.Handler) http.Handler

// Router wraps http.ServeMux with middleware support
type Router struct {
	mux        *http.ServeMux
	middleware []Middleware
	logger     *slog.Logger
}

// NewRouter creates a new Router instance
func NewRouter(logger *slog.Logger) *Router {
	return &Router{
		mux:    http.NewServeMux(),
		logger: logger,
	}
}

// Use adds middleware to the router
func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

// Handle registers a handler for the given pattern with all middleware applied
func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	// Wrap handler with all middleware in reverse order
	// This ensures middleware executes in the order they were added
	var h http.Handler = handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	r.mux.Handle(pattern, h)
}

// HandleFunc is a convenience method that converts a function to HandlerFunc
func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.Handle(pattern, http.HandlerFunc(handler))
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// GET is a convenience method for GET requests
func (r *Router) GET(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, req)
	})
}

// POST is a convenience method for POST requests
func (r *Router) POST(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, req)
	})
}

// PUT is a convenience method for PUT requests
func (r *Router) PUT(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, req)
	})
}

// DELETE is a convenience method for DELETE requests
func (r *Router) DELETE(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, req)
	})
}

// OPTIONS is a convenience method for OPTIONS requests
func (r *Router) OPTIONS(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodOptions {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, req)
	})
}
