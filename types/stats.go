package types

// List Stats

type ListStats struct {
	TotalBots          int64 `json:"total_bots" description:"The list of all bots on the list as ListStatsBot objects (partial bot objects)"`
	TotalApprovedBots  int64 `json:"total_approved_bots" description:"The total number of approved bots on the list"`
	TotalCertifiedBots int64 `json:"total_certified_bots" description:"The total number of certified bots on the list"`
	TotalStaff         int64 `json:"total_staff" description:"The total number of staff members on the list"`
	TotalUsers         int64 `json:"total_users" description:"The total number of users on the list"`
	TotalVotes         int64 `json:"total_votes" description:"The total number of votes on the list"`
	TotalPacks         int64 `json:"total_packs" description:"The total number of packs on the list"`
	TotalTickets       int64 `json:"total_tickets" description:"The total number of tickets created on the list"`
}

type StatusDocs struct {
	Key1 string `json:"key1" description:"Some key-value pairs from our status API"`
	Key2 string `json:"key2" description:"Some key-value pairs from our status API"`
	Key3 string `json:"key3" description:"Some key-value pairs from our status API"`
	Etc  string `json:"etc" description:"And so on..."`
}

type StaffTeam struct {
	Members []UserPerm `json:"members"`
}
