package web_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ecnepsnai/web"
)

func TestHTTPAddRoutes(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	options := web.HandleOptions{}

	server.HTTP.GET("/"+randomString(5), handle, options)
	server.HTTP.HEAD("/"+randomString(5), handle, options)
	server.HTTP.GETHEAD("/"+randomString(5), handle, options)
	server.HTTP.OPTIONS("/"+randomString(5), handle, options)
	server.HTTP.POST("/"+randomString(5), handle, options)
	server.HTTP.PUT("/"+randomString(5), handle, options)
	server.HTTP.PATCH("/"+randomString(5), handle, options)
	server.HTTP.DELETE("/"+randomString(5), handle, options)
}

func TestHTTPAuthenticated(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	authenticate := func(request *http.Request) interface{} {
		return 1
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPUnauthenticated(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	authenticate := func(request *http.Request) interface{} {
		return nil
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 401 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 401, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPNotFound(t *testing.T) {
	t.Parallel()
	server := newServer()

	path := randomString(5)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 404 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 404, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPMethodNotAllowed(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	authenticate := func(request *http.Request) interface{} {
		return nil
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	path := randomString(5)

	server.HTTP.POST("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 405 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 405, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPHandleError(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{
			Status: 403,
		}
	}
	authenticate := func(request *http.Request) interface{} {
		return 1
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 403 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 403, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPResponse(t *testing.T) {
	t.Parallel()
	server := newServer()

	tmp := t.TempDir()
	data := randomString(5)
	name := randomString(5) + ".html"

	if err := os.WriteFile(path.Join(tmp, name), []byte(data), 0644); err != nil {
		t.Fatalf("Error making temporary file: %s", err.Error())
	}

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		f, err := os.Open(path.Join(tmp, name))
		if err != nil {
			t.Fatalf("Error opening temporary file: %s", err.Error())
		}
		return web.HTTPResponse{
			Reader: f,
		}
	}
	options := web.HandleOptions{}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPContentType(t *testing.T) {
	t.Parallel()
	server := newServer()

	contentType := "application/amazing"
	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{
			ContentType: contentType,
		}
	}
	options := web.HandleOptions{}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != contentType {
		t.Fatalf("Unexpected content type. Expected '%s' got '%s'", contentType, resp.Header.Get("Content-Type"))
	}
}

func TestHTTPHeaders(t *testing.T) {
	t.Parallel()
	server := newServer()

	headerKey := randomString(5)
	headerValue := randomString(5)
	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{
			Headers: map[string]string{
				headerKey: headerValue,
			},
		}
	}
	options := web.HandleOptions{}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	if resp.Header.Get(headerKey) != headerValue {
		t.Fatalf("Unexpected content type. Expected '%s: %s' got '%s: %s'", headerKey, headerValue, headerKey, resp.Header.Get(headerKey))
	}
}

func TestServeFile(t *testing.T) {
	t.Parallel()
	server := newServer()

	tmp := t.TempDir()
	data := randomString(5)
	name := randomString(5) + ".html"

	if err := os.WriteFile(path.Join(tmp, name), []byte(data), 0644); err != nil {
		t.Fatalf("Error making temporary file: %s", err.Error())
	}

	server.HTTP.Static("/", tmp)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, name))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPUnauthorizedMethod(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	authenticate := func(request *http.Request) interface{} {
		return nil
	}

	location := "somewhere-else"

	unauthorized := func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Location", location)
		w.WriteHeader(410)
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
		UnauthorizedMethod: unauthorized,
	}

	path := randomString(5)

	server.HTTP.GET("/"+path, handle, options)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 410 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 410, resp.StatusCode)
	}
	if resp.Header.Get("Location") != location {
		t.Fatalf("Missing expected HTTP header. Expected '%s' got '%s'", location, resp.Header.Get("Location"))
	}
}

func TestHTTPLargeBody(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	options := web.HandleOptions{
		MaxBodyLength: 10,
	}

	path := randomString(5)
	body := bytes.NewReader([]byte(randomString(50)))

	server.HTTP.POST("/"+path, handle, options)

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path), "text-plain", body)
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 413 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 413, resp.StatusCode)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
}

func TestHTTPRateLimit(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{}
	}
	options := web.HandleOptions{}

	path := randomString(5)

	server.Options.MaxRequestsPerSecond = 2
	server.HTTP.GET("/"+path, handle, options)

	testIdx := 1
	doTest := func(expectedStatus int) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
		if err != nil {
			t.Fatalf("Network error: %s", err.Error())
		}
		if resp.StatusCode != expectedStatus {
			t.Fatalf("Unexpected HTTP status code. Expected %d got %d in test %d", expectedStatus, resp.StatusCode, testIdx)
		}
		resp.Body.Close()
		testIdx++
	}

	doTest(200)
	time.Sleep(500 * time.Millisecond)
	doTest(200)
	time.Sleep(500 * time.Millisecond)
	doTest(200)
	time.Sleep(500 * time.Millisecond)
	doTest(200)
	doTest(200)
	doTest(429)
	time.Sleep(1 * time.Second)

	doTest(200)
	doTest(200)
	doTest(429)
}

func TestHTTPGETHEAD(t *testing.T) {
	t.Parallel()
	server := newServer()

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		data := []byte("Hello, world!")
		return web.HTTPResponse{
			Reader:        io.NopCloser(bytes.NewReader(data)),
			ContentType:   "text/plain",
			ContentLength: uint64(len(data)),
		}
	}

	path := randomString(5)

	server.HTTP.GETHEAD("/"+path, handle, web.HandleOptions{})

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
	if len(data) == 0 {
		t.Fatalf("No data returned when expected")
	}
	data = nil

	resp, err = http.Head(fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path))
	if err != nil {
		t.Fatalf("Network error: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 200, resp.StatusCode)
	}
	data, _ = io.ReadAll(resp.Body)
	if len(data) > 0 {
		t.Fatalf("Data returned when none expected: %s", data)
	}
}

type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }

func TestHTTPRangeGet(t *testing.T) {
	t.Parallel()
	server := newServer()

	rawData := make([]byte, 250)
	randomData := make([]byte, 500)
	rand.Read(rawData)
	hex.Encode(randomData, rawData)
	reader := nopSeekCloser{bytes.NewReader(randomData)}
	if len(randomData) != 500 {
		panic("Not enough random data?")
	}

	handle := func(request web.Request, writer web.Writer) web.HTTPResponse {
		return web.HTTPResponse{
			Reader:        reader,
			ContentType:   "text/plain",
			ContentLength: 500,
		}
	}

	path := randomString(5)

	server.HTTP.GETHEAD("/"+path, handle, web.HandleOptions{})

	url := fmt.Sprintf("http://localhost:%d/%s", server.ListenPort, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("range", "bytes=0-99,200-300,400-")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 206 {
		t.Fatalf("Unexpected status code for URL '%s'. Expected %d got %d", url, 206, resp.StatusCode)
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		panic(err)
	}
	r := multipart.NewReader(resp.Body, params["boundary"])
	i := 0

	ranges := []string{
		"bytes 0-99/500",
		"bytes 200-300/500",
		"bytes 400-499/500",
	}

	data := [][]byte{
		randomData[0:100],
		randomData[200:301],
		randomData[400:],
	}

	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if i > 2 {
			t.Fatalf("Unpexted number of unit parts in response. Expected 3 but got at least %d", i+1)
		}
		contentType := part.Header.Get("Content-Type")
		contentRange := part.Header.Get("Content-Range")
		if contentType != "text/plain" {
			t.Errorf("Unexpected content type in unit part %d. Expected %s got %s", i+1, "text/plain", contentType)
		}
		if contentRange != ranges[i] {
			t.Errorf("Unexpected content range in unit part %d. Expected %s got %s", i+1, ranges[i], contentRange)
		}
		partData, err := io.ReadAll(part)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(partData, data[i]) {
			t.Errorf("Unexpected data in unit part %d.\nExpected:\n\t%s\nGot:\n\t%s", i+1, data[i], partData)
		}
		i++
	}
}
