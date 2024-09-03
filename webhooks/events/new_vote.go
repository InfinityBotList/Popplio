package events

import (
	"popplio/validators"
	"popplio/webhooks/core/events"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookNewVoteData struct {
	Votes    int  `json:"votes" description:"The number of votes the entity received"`
	PerUser  int  `json:"per_user" description:"The number of votes the user has given to the entity. For example, if the vote is a double vote (weekend), this would be 2."`
	Downvote bool `json:"downvote" description:"Whether the vote was a downvote"`
}

func (v WebhookNewVoteData) TargetTypes() []string {
	return []string{
		"bot",
		"server",
		"team",
	}
}

func (v WebhookNewVoteData) Event() string {
	return "NEW_VOTE"
}

func (v WebhookNewVoteData) Summary() string {
	return "New Vote"
}

func (v WebhookNewVoteData) Description() string {
	return "This webhook is sent when a user votes for an entity."
}

func (v WebhookNewVoteData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {
	return &discord.Embed{
		URL: "https://botlist.site/" + targets.GetID(),
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       "🎉 Vote Count Updated!",
		Description: ":heart: " + creator.DisplayName + " has voted for " + targets.GetTargetName(),
		Color:       0x8A6BFD,
		Fields: []discord.EmbedField{
			{
				Name:   "Vote Count:",
				Value:  strconv.Itoa(int(v.Votes)),
				Inline: validators.TruePtr,
			},
			{
				Name: "Downvote:",
				Value: func() string {
					if v.Downvote {
						return "Yes"
					}
					return "No"
				}(),
			},
			{
				Name:   "User ID:",
				Value:  creator.ID,
				Inline: validators.TruePtr,
			},
			{
				Name:   "View Page",
				Value:  targets.GetViewLink(),
				Inline: validators.TruePtr,
			},
		},
	}
}
func init() {
	events.AddEvent(WebhookNewVoteData{})
}
