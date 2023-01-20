package get_user

import (
	"popplio/api"
	"testing"
)

func TestGetUser(t *testing.T) {
	api.Test(api.TestData{
		Route: Route,
		Body:  []byte{},
		T:     t,
		Params: map[string]string{
			"id": "510065483693817867",
		},
		AuthID: "510065483693817867",
	})
}
