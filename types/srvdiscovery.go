package types

type ServiceDiscovery struct {
	Services map[string]SDService `json:"services"`
}

type SDService struct {
	Url                string `json:"url"`
	Description        string `json:"description"`
	PlannedMaintenance bool   `json:"planned_maintenance"`
	NeedsStaging       bool   `json:"needs_staging"`
}
