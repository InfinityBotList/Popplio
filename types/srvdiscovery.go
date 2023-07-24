package types

type ServiceDiscovery struct {
	Services map[string]SDService `json:"services"`
}

type SDService struct {
	Url          string `json:"url,omitempty"`
	Docs         string `json:"docs,omitempty"`
	Description  string `json:"description"`
	NeedsStaging bool   `json:"needs_staging"`
}
