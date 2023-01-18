package get_announcements

import (
	"popplio/api"
	"testing"
)

func TestGetAnnouncements(t *testing.T) {
	api.Test(api.TestData{
		Route:  Route,
		Body:   []byte{},
		T:      t,
		Params: map[string]string{},
	})
}
