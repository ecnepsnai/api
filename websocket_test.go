package web_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ecnepsnai/web"
	"github.com/gorilla/websocket"
)

func TestWebsocketAuthenticated(t *testing.T) {
	t.Parallel()
	server := newServer()

	authenticate := func(request *http.Request) interface{} {
		return 1
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	type questionType struct {
		Name string `json:"name"`
	}

	type answerType struct {
		Greeting string `json:"greeting"`
	}

	server.Socket("/socket", func(request web.Request, conn *web.WSConn) {
		defer conn.Close()

		question := questionType{}
		if err := conn.ReadJSON(&question); err != nil {
			t.Errorf("Error reading question JSON: %s", err.Error())
			return
		}

		answer := answerType{
			Greeting: question.Name,
		}
		if err := conn.WriteJSON(&answer); err != nil {
			t.Errorf("Error writing answer JSON: %s", err.Error())
			return
		}
	}, options)

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d/socket", server.ListenPort), nil)
	if err != nil {
		t.Fatalf("Error connecting to websocket: %s", err.Error())
	}

	question := questionType{Name: randomString(6)}
	if err := conn.WriteJSON(&question); err != nil {
		t.Fatalf("Error sending JSON message to server: %s", err.Error())
	}

	answer := answerType{}
	if err := conn.ReadJSON(&answer); err != nil {
		t.Errorf("Error reading answer JSON: %s", err.Error())
		return
	}

	if answer.Greeting != question.Name {
		t.Errorf("Unexpected response. Expected '%s' got '%s'", question.Name, answer.Greeting)
	}
}

func TestWebsocketUnauthenticated(t *testing.T) {
	t.Parallel()
	server := newServer()

	authenticate := func(request *http.Request) interface{} {
		return nil
	}
	options := web.HandleOptions{
		AuthenticateMethod: authenticate,
	}

	server.Socket("/socket", func(request web.Request, conn *web.WSConn) {
		conn.Close()
	}, options)

	_, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d/socket", server.ListenPort), nil)
	if err != nil && !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("Error connecting to websocket: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 401 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 401, resp.StatusCode)
	}
}

func TestWebsocketPanic(t *testing.T) {
	t.Parallel()
	server := newServer()

	options := web.HandleOptions{}

	server.Socket("/socket", func(request web.Request, conn *web.WSConn) {
		panic("Oh no!")
	}, options)

	_, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d/socket", server.ListenPort), nil)
	if err != nil && !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("Error connecting to websocket: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
}

func TestWebsocketPreHandle(t *testing.T) {
	t.Parallel()
	server := newServer()

	path200 := randomString(5)
	path400 := randomString(5)

	type questionType struct {
		Name string `json:"name"`
	}

	type answerType struct {
		Greeting string `json:"greeting"`
	}
	handle := func(request web.Request, conn *web.WSConn) {
		defer conn.Close()

		question := questionType{}
		if err := conn.ReadJSON(&question); err != nil {
			t.Errorf("Error reading question JSON: %s", err.Error())
			return
		}

		answer := answerType{
			Greeting: question.Name,
		}
		if err := conn.WriteJSON(&answer); err != nil {
			t.Errorf("Error writing answer JSON: %s", err.Error())
			return
		}
	}

	options := web.HandleOptions{
		PreHandle: func(w http.ResponseWriter, request *http.Request) error {
			if request.URL.Path == "/"+path400 {
				w.WriteHeader(400)
				return fmt.Errorf("boo")
			}
			return nil
		},
	}

	server.Socket("/"+path200, handle, options)
	server.Socket("/"+path400, handle, options)

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d/"+path200, server.ListenPort), nil)
	if err != nil {
		t.Fatalf("Error connecting to websocket: %s", err.Error())
	}

	question := questionType{Name: randomString(6)}
	if err := conn.WriteJSON(&question); err != nil {
		t.Fatalf("Error sending JSON message to server: %s", err.Error())
	}

	answer := answerType{}
	if err := conn.ReadJSON(&answer); err != nil {
		t.Errorf("Error reading answer JSON: %s", err.Error())
		return
	}

	if answer.Greeting != question.Name {
		t.Errorf("Unexpected response. Expected '%s' got '%s'", question.Name, answer.Greeting)
	}

	_, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d/"+path400, server.ListenPort), nil)
	if err != nil && !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("Error connecting to websocket: %s", err.Error())
	}
	if resp == nil {
		t.Fatalf("Nil response returned")
	}
	if resp.StatusCode != 400 {
		t.Fatalf("Unexpected HTTP status code. Expected %d got %d", 400, resp.StatusCode)
	}
}
