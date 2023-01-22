package types

type StaffOnboardCode struct {
	Code string `json:"code"`
}

type StaffOnboardData struct {
	UserID string         `json:"user_id"`
	Data   map[string]any `json:"data"`
}
