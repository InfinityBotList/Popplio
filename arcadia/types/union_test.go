package types

import (
	"encoding/json"
	"testing"
	"time"
)

// The wire format is frozen. Every case below is the exact JSON the panel sends
// or expects; if one of these changes, the panel breaks silently.
func TestUnionEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value json.Marshaler
		json  string
	}{
		{
			name:  "struct variant",
			value: PanelQuery{Hello: &QHello{LoginToken: "abc", Version: 5}},
			json:  `{"Hello":{"login_token":"abc","version":5}}`,
		},
		{
			name: "nested unit variant as bare string",
			value: PanelQuery{UpdateStaffPositions: &QUpdateStaffPositions{
				LoginToken: "tok",
				Action:     StaffPositionAction{ListPositions: unitSet()},
			}},
			json: `{"UpdateStaffPositions":{"login_token":"tok","action":"ListPositions"}}`,
		},
		{
			name: "nested struct variant",
			value: PanelQuery{UpdateStaffPositions: &QUpdateStaffPositions{
				LoginToken: "tok",
				Action:     StaffPositionAction{SetIndex: &StaffSetIndex{ID: "x", Index: 3}},
			}},
			json: `{"UpdateStaffPositions":{"login_token":"tok","action":{"SetIndex":{"id":"x","index":3}}}}`,
		},
		{
			name: "target type is a variant name, rpc method is an object",
			value: PanelQuery{ExecuteRpc: &QExecuteRpc{
				LoginToken: "tok",
				TargetType: TargetTypeBot,
				Method:     RPCMethod{Claim: &RPCClaim{TargetID: "1", Force: true}},
			}},
			json: `{"ExecuteRpc":{"login_token":"tok","target_type":"Bot","method":{"Claim":{"target_id":"1","force":true}}}}`,
		},
		{
			name: "authorize begin",
			value: PanelQuery{Authorize: &QAuthorize{
				Version: 5,
				Action:  AuthorizeAction{Begin: &AuthBegin{Scope: "s", RedirectURL: "r"}},
			}},
			json: `{"Authorize":{"version":5,"action":{"Begin":{"scope":"s","redirect_url":"r"}}}}`,
		},
		{
			name:  "partner unit variant",
			value: PartnerAction{List: unitSet()},
			json:  `"List"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if string(got) != tt.json {
				t.Errorf("encode mismatch\n got: %s\nwant: %s", got, tt.json)
			}
		})
	}
}

// chrono prints 0, 3, 6 or 9 fractional digits; Go's own marshaller trims
// trailing zeroes one at a time and would emit ".5" where chrono emits ".500".
func TestTimestampMatchesChrono(t *testing.T) {
	tests := []struct {
		nanos int
		want  string
	}{
		{0, "2024-05-26T05:16:06Z"},
		{500_000_000, "2024-05-26T05:16:06.500Z"},
		{574_978_000, "2024-05-26T05:16:06.574978Z"},
		{574_978_123, "2024-05-26T05:16:06.574978123Z"},
	}

	for _, tt := range tests {
		ts := NewTimestamp(time.Date(2024, 5, 26, 5, 16, 6, tt.nanos, time.UTC))

		if got := ts.Format(); got != tt.want {
			t.Errorf("Format() = %s, want %s", got, tt.want)
		}
	}
}

func TestTargetTypeHasTwoStringForms(t *testing.T) {
	if got := TargetTypeBot.Variant(); got != "Bot" {
		t.Errorf("Variant() = %q, want %q", got, "Bot")
	}

	if got := TargetTypeBot.String(); got != "bot" {
		t.Errorf("String() = %q, want %q", got, "bot")
	}
}
