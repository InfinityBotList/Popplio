package get_announcements

import (
	"popplio/api"
	"testing"
)

func TestGetAnnouncements(t *testing.T) {
	api.Test(Route, []byte{}, t)
}
