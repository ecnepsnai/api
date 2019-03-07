package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

// API describes a JSON API
type API struct {
	server *Server
}

// GET register a new HTTP GET request handle
func (a API) GET(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("GET", path, handle, options)
}

// HEAD register a new HTTP HEAD request handle
func (a API) HEAD(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("HEAD", path, handle, options)
}

// OPTIONS register a new HTTP OPTIONS request handle
func (a API) OPTIONS(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("OPTIONS", path, handle, options)
}

// POST register a new HTTP POST request handle
func (a API) POST(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("POST", path, handle, options)
}

// PUT register a new HTTP PUT request handle
func (a API) PUT(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("PUT", path, handle, options)
}

// PATCH register a new HTTP PATCH request handle
func (a API) PATCH(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("PATCH", path, handle, options)
}

// DELETE register a new HTTP DELETE request handle
func (a API) DELETE(path string, handle APIHandle, options HandleOptions) {
	a.registerAPIEndpoint("DELETE", path, handle, options)
}

func (a API) registerAPIEndpoint(method string, path string, handle APIHandle, options HandleOptions) {
	a.server.log.Debug("Register API %s %s", method, path)
	a.server.router.Handle(method, path, a.apiPreHandle(handle, options))
}

func (a API) apiPreHandle(endpointHandle APIHandle, options HandleOptions) httprouter.Handle {
	return func(w http.ResponseWriter, request *http.Request, ps httprouter.Params) {
		if options.MaxBodyLength > 0 {
			// We don't need to worry about this not being a number. Go's own HTTP server
			// won't respond to requests like these
			length, _ := strconv.ParseUint(request.Header.Get("Content-Length"), 10, 64)
			a.server.log.Debug("Body length: %d", length)

			if length > options.MaxBodyLength {
				a.server.log.Error("Rejecting HTTP request with body length %d", length)
				w.WriteHeader(413)
				return
			}
		}

		if options.AuthenticateMethod != nil {
			userData := options.AuthenticateMethod(request)
			if isUserdataNil(userData) {
				if options.UnauthorizedMethod == nil {
					a.server.log.Warn("Rejected authenticated request")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(Error{401, "Unauthorized"})
					return
				}

				options.UnauthorizedMethod(w, request)
			} else {
				a.apiPostHandle(endpointHandle, userData)(w, request, ps)
			}
		}
		a.apiPostHandle(endpointHandle, nil)(w, request, ps)
	}
}

func (a API) apiPostHandle(endpointHandle APIHandle, userData interface{}) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")

		response := JSONResponse{}
		request := Request{
			Writer:   w,
			HTTP:     r,
			Params:   ps,
			UserData: userData,
			log:      a.server.log,
		}

		start := time.Now()
		data, err := endpointHandle(request)
		elapsed := time.Since(start)
		if err != nil {
			response.Code = err.Code
			w.WriteHeader(err.Code)
			response.Error = *err
		} else {
			response.Code = 200
			response.Data = data
		}
		a.server.log.Info("API Request: %s %s -> %d (%s)", r.Method, r.RequestURI, response.Code, elapsed)
		json.NewEncoder(w).Encode(response)
	}
}

type notFoundHandler struct {
	server *Server
}

func (n notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.server.log.Info("HTTP %s %s -> %d", r.Method, r.RequestURI, 404)
	w.WriteHeader(404)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Error{404, "Not found"})
}

type methodNotAllowedHandler struct {
	server *Server
}

func (n methodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.server.log.Info("HTTP %s %s -> %d", r.Method, r.RequestURI, 405)
	w.WriteHeader(405)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Error{405, "Method not allowed"})
}
