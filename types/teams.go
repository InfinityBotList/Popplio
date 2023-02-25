package types

type Team struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Avatar    string       `json:"avatar"`
	MainOwner *DiscordUser `json:"main_owner"`
	Members   []TeamMember `json:"members"`
}

type TeamMember struct {
	User  *DiscordUser `json:"user"`
	Perms []string     `json:"perms"`
}
