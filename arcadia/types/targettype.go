package types

import (
	"encoding/json"
	"fmt"
)

// TargetType is the kind of entity an action applies to.
//
// It has TWO distinct string forms and mixing them up silently corrupts data:
//
//	JSON / FromStr : "Bot", "Server", "Team", "Pack", "User"   (variant names)
//	Display        : "bot", "server", "team", "pack", "user"   (lowercase)
//
// The Display form is what goes into SQL (entity_votes.target_type) and into
// Discord embed fields. The JSON form is what the panel sends and receives.
type TargetType string

const (
	TargetTypeBot    TargetType = "Bot"
	TargetTypeServer TargetType = "Server"
	TargetTypeTeam   TargetType = "Team"
	TargetTypePack   TargetType = "Pack"
	TargetTypeUser   TargetType = "User"
)

// TargetTypeVariants is in declaration order, which is the order the Hello
// response's target_types array must use.
var TargetTypeVariants = []TargetType{
	TargetTypeBot,
	TargetTypeServer,
	TargetTypeTeam,
	TargetTypePack,
	TargetTypeUser,
}

// String is the Display impl: lowercase. This is the SQL/embed form.
func (t TargetType) String() string {
	switch t {
	case TargetTypeBot:
		return "bot"
	case TargetTypeServer:
		return "server"
	case TargetTypeTeam:
		return "team"
	case TargetTypePack:
		return "pack"
	case TargetTypeUser:
		return "user"
	default:
		return string(t)
	}
}

// Variant is the JSON/FromStr form: the variant name.
func (t TargetType) Variant() string {
	return string(t)
}

func (t TargetType) Valid() bool {
	switch t {
	case TargetTypeBot, TargetTypeServer, TargetTypeTeam, TargetTypePack, TargetTypeUser:
		return true
	default:
		return false
	}
}

// TargetTypeFromString mirrors strum's FromStr, which matches on variant names.
func TargetTypeFromString(s string) (TargetType, error) {
	t := TargetType(s)
	if !t.Valid() {
		return "", fmt.Errorf("Matching variant not found")
	}

	return t, nil
}

// ID is the primary key column name for this entity type.
func (t TargetType) ID() string {
	switch t {
	case TargetTypeBot:
		return "bot_id"
	case TargetTypeServer:
		return "server_id"
	case TargetTypeTeam:
		return "id"
	case TargetTypePack:
		return "url"
	case TargetTypeUser:
		return "user_id"
	default:
		return ""
	}
}

func (t TargetType) SupportsVotes() bool {
	switch t {
	case TargetTypeBot, TargetTypeServer, TargetTypeTeam, TargetTypePack:
		return true
	default:
		return false
	}
}

func (t *TargetType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := TargetTypeFromString(s)
	if err != nil {
		return fmt.Errorf("unknown variant `%s` for TargetType", s)
	}

	*t = parsed
	return nil
}

func (t TargetType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}
