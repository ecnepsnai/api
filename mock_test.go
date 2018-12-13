package web

import (
	"testing"
)

func TestMock(t *testing.T) {
	type exampleType struct {
		Enabled bool
	}

	userData := 1

	handle := func(request Request) (interface{}, *Error) {
		example := exampleType{}

		if err := request.Decode(&example); err != nil {
			t.Error("Error decoding example JSON object from mocked request")
		}
		if !example.Enabled {
			t.Error("Invalid HTTP body from mocked request")
		}

		if request.UserData.(int) != userData {
			t.Error("Invalid user data")
		}

		if request.Params.ByName("foo") != "bar" {
			t.Error("Invalid request path parameters")
		}

		return nil, nil
	}

	request := MockRequest(userData, map[string]string{"foo": "bar"}, exampleType{true})
	handle(request)
}
