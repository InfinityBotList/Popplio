package validators

import "strings"

// This function normalizes the target type to its correct form.
func NormalizeTargetType(targetType string) string {
	switch targetType {
	// Bot
	case "bots":
		return "bot"
	// User
	case "users":
		return "user"
	case "user":
		return "user"
	// Server
	case "servers":
		return "server"
	case "server":
		return "server"
	// Teams
	case "teams":
		return "team"
	case "team":
		return "team"
	// Packs
	case "packs":
		return "pack"
	case "pack":
		return "pack"
	default:
		return strings.TrimSuffix(targetType, "s")
	}
}
