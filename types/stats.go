package types

// List Stats
type ListStatsBot struct {
	BotID     string `json:"bot_id" description:"The bot's ID"`
	Vanity    string `json:"vanity" description:"The bot's vanity URL if it has one, otherwise null"`
	Short     string `json:"short" description:"The bot's short description"`
	Type      string `json:"type" description:"The bot's type (e.g. pending/approved/certified/denied  etc.)"`
	QueueName string `json:"queue_name" description:"The bot's queue name if it has one, otherwise null"`
}

type ListStats struct {
	Bots         []ListStatsBot `json:"bots" description:"The list of all bots on the list as ListStatsBot objects (partial bot objects)"`
	TotalStaff   int64          `json:"total_staff" description:"The total number of staff members on the list"`
	TotalUsers   int64          `json:"total_users" description:"The total number of users on the list"`
	TotalVotes   int64          `json:"total_votes" description:"The total number of votes on the list"`
	TotalPacks   int64          `json:"total_packs" description:"The total number of packs on the list"`
	TotalTickets int64          `json:"total_tickets" description:"The total number of tickets created on the list"`
}
