package get_staff_permissions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"popplio/perms"
	"popplio/types"

	"github.com/infinitybotlist/eureka/uapi"
)

// The response is what a permission picker is built from, so it has to carry
// every permission a role can hold, with enough on each one to render it.
func TestRouteListsTheWholeCatalogue(t *testing.T) {
	resp := Route(uapi.RouteData{}, httptest.NewRequest(http.MethodGet, "/staff/meta/permissions", nil))

	body, ok := resp.Json.(types.PermissionResponse)

	if !ok {
		t.Fatalf("response is %T, want types.PermissionResponse", resp.Json)
	}

	defs := perms.Staff.Definitions()

	if len(body.Perms) != len(defs) {
		t.Fatalf("returned %d permissions, catalogue has %d", len(body.Perms), len(defs))
	}

	for i, p := range body.Perms {
		if p.ID != string(defs[i].ID) {
			t.Errorf("permission %d is %q, want %q — catalogue order must be preserved", i, p.ID, defs[i].ID)
		}

		if p.Name == "" || p.Desc == "" || p.Category == "" {
			t.Errorf("%s is missing a name, description or category", p.ID)
		}
	}

	// The entity catalogue is a different domain and must not leak in.
	ids := make([]string, 0, len(body.Perms))

	for _, p := range body.Perms {
		ids = append(ids, p.ID)
	}

	if slices.Contains(ids, string(perms.EntityAddBots)) {
		t.Error("the staff endpoint returned an entity permission")
	}

	for _, want := range []string{"administrator", "review_bots", "manage_staff_roles"} {
		if !slices.Contains(ids, want) {
			t.Errorf("catalogue is missing %q", want)
		}
	}
}

// The dangerous flag is what lets a client warn before handing over something
// irreversible, so it has to survive serialization.
func TestDangerousSurvivesJSON(t *testing.T) {
	resp := Route(uapi.RouteData{}, httptest.NewRequest(http.MethodGet, "/staff/meta/permissions", nil))

	encoded, err := json.Marshal(resp.Json)

	if err != nil {
		t.Fatalf("marshalling the response: %v", err)
	}

	var decoded struct {
		Perms []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Desc      string `json:"desc"`
			Category  string `json:"category"`
			Dangerous bool   `json:"dangerous"`
		} `json:"perms"`
	}

	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decoding the response: %v", err)
	}

	var dangerous, safe bool

	for _, p := range decoded.Perms {
		switch p.ID {
		case string(perms.StaffForceRemoveBots):
			dangerous = p.Dangerous
		case string(perms.StaffViewPanel):
			safe = !p.Dangerous
		}
	}

	if !dangerous {
		t.Error("force_remove_bots should be flagged dangerous")
	}

	if !safe {
		t.Error("view_panel should not be flagged dangerous")
	}
}
