package types

type ServiceDiscovery struct {
	Services []SDService `json:"services"`
}

type SDService struct {
	ID           string `json:"id"`
	ProdURL      string `json:"prod_url"`
	StagingURL   string `json:"staging_url,omitempty"`
	Docs         string `json:"docs,omitempty"`
	Description  string `json:"description"`
	NeedsStaging bool   `json:"needs_staging,omitempty"`
}

type SDList struct {
	Services []string `json:"services"`
}
