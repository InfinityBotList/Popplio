package types

type ServiceDiscovery struct {
	Services []SDService `json:"services"`
}

type SDService struct {
	ID           string `json:"id"`
	Url          string `json:"url,omitempty"`
	Docs         string `json:"docs,omitempty"`
	Description  string `json:"description"`
	NeedsStaging bool   `json:"needs_staging"`
}
