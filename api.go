package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/ecnepsnai/web/router"
)

// API describes a JSON API server. API handles return data or an error, and all responses are wrapped in a common
// response object; [web.JSONResponse].
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
	log.PDebug("Register API endpoint", map[string]interface{}{
		"method": method,
		"path":   path,
	})
	a.server.router.Handle(method, path, a.apiPreHandle(handle, options))
}

func (a API) apiPreHandle(endpointHandle APIHandle, options HandleOptions) router.Handle {
	return func(w http.ResponseWriter, request router.Request) {
		if options.PreHandle != nil {
			if err := options.PreHandle(w, request.HTTP); err != nil {
				return
			}
		}

		if a.server.isRateLimited(w, request.HTTP) {
			return
		}

		if options.MaxBodyLength > 0 {
			// We don't need to worry about this not being a number. Go's own HTTP server
			// won't respond to requests like these
			length, _ := strconv.ParseUint(request.HTTP.Header.Get("Content-Length"), 10, 64)

			if length > options.MaxBodyLength {
				log.PError("Rejecting API request with oversized body", map[string]interface{}{
					"body_length": length,
					"max_length":  options.MaxBodyLength,
				})
				w.WriteHeader(413)
				return
			}
		}

		if options.AuthenticateMethod != nil {
			userData := options.AuthenticateMethod(request.HTTP)
			if isUserdataNil(userData) {
				if options.UnauthorizedMethod == nil {
					log.PWarn("Rejected request to authenticated API endpoint", map[string]interface{}{
						"url":         request.HTTP.URL,
						"method":      request.HTTP.Method,
						"remote_addr": RealRemoteAddr(request.HTTP),
					})
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(Error{401, "Unauthorized"})
					return
				}

				options.UnauthorizedMethod(w, request.HTTP)
			} else {
				a.apiPostHandle(endpointHandle, userData, options)(w, request)
			}
			return
		}
		a.apiPostHandle(endpointHandle, nil, options)(w, request)
	}
}

func (a API) apiPostHandle(endpointHandle APIHandle, userData interface{}, options HandleOptions) router.Handle {
	return func(w http.ResponseWriter, r router.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := JSONResponse{}
		request := Request{
			HTTP:       r.HTTP,
			Parameters: r.Parameters,
			UserData:   userData,
		}

		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				log.PError("Recovered from panic during API handle", map[string]interface{}{
					"error":  fmt.Sprintf("%v", p),
					"route":  r.HTTP.URL.Path,
					"method": r.HTTP.Method,
					"stack":  string(debug.Stack()),
				})
				w.WriteHeader(500)
				json.NewEncoder(w).Encode(JSONResponse{Error: CommonErrors.ServerError})
			}
		}()

		data, resp, err := endpointHandle(request)
		if resp != nil {
			for key, value := range resp.Headers {
				w.Header().Set(key, value)
			}
			for _, cookie := range resp.Cookies {
				http.SetCookie(w, &cookie)
			}
		}

		elapsed := time.Since(start)
		if err != nil {
			w.WriteHeader(err.Code)
			response.Error = err
		} else {
			response.Data = data
		}
		if !options.DontLogRequests {
			log.PWrite(a.server.Options.RequestLogLevel, "API Request", map[string]interface{}{
				"remote_addr": RealRemoteAddr(r.HTTP),
				"method":      r.HTTP.Method,
				"url":         r.HTTP.URL,
				"elapsed":     elapsed.String(),
			})
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			if strings.Contains(err.Error(), "write: broken pipe") {
				return
			}

			log.PError("Error writing response", map[string]interface{}{
				"method": r.HTTP.Method,
				"url":    r.HTTP.URL,
				"error":  err.Error(),
			})
		}
	}
}
