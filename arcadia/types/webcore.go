package types

// InstanceConfig describes the panel instance to the frontend.
type InstanceConfig struct {
	Description string   `json:"description"`
	Warnings    []string `json:"warnings"`
}

// CoreConstants are the URLs and guild ids the panel needs. Guild ids are
// STRINGS here even though they are u64 in config.
//
// NOTE: upstream also carried htmlsanitize_url and cdn_url. Both were removed
// along with the services behind them, so neither key appears in the Hello
// response any more. See CONFORMANCE.md.
type CoreConstants struct {
	FrontendURL    string       `json:"frontend_url"`
	InfernoplexURL string       `json:"infernoplex_url"`
	PopplioURL     string       `json:"popplio_url"`
	Servers        PanelServers `json:"servers"`
}

type PanelServers struct {
	Main    string `json:"main"`
	Staff   string `json:"staff"`
	Testing string `json:"testing"`
}

// StartAuth is the Authorize/Begin response.
type StartAuth struct {
	LoginURL      string `json:"login_url"`
	Scope         string `json:"scope"`
	ResponseScope string `json:"response_scope"`
}

// Hello is the session bootstrap payload.
type Hello struct {
	InstanceConfig InstanceConfig `json:"instance_config"`
	AuthData       AuthData       `json:"auth_data"`
	StaffMember    StaffMember    `json:"staff_member"`
	CoreConstants  CoreConstants  `json:"core_constants"`
	TargetTypes    []TargetType   `json:"target_types"`
}
