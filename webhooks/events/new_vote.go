package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookNewVoteData struct {
	Votes    int  `json:"votes" description:"The number of votes the entity received"`
	Downvote bool `json:"downvote" description:"Whether the vote was a downvote"`
	PerUser  int  `json:"per_user" description:"The number of votes the user has given"`
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

func (v WebhookNewVoteData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.GetID(),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.GetAvatarURL(),
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart: " + creator.DisplayName + " has voted for " + targets.GetTargetName(),
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Vote Count:",
						Value:  strconv.Itoa(int(v.Votes)),
						Inline: true,
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
						Inline: true,
					},
					{
						Name:   "View Page",
						Value:  targets.GetViewLink(),
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  targets.GetTargetLink("Vote for", "/vote"),
						Inline: true,
					},
				},
			},
		},
	}
}
func init() {
	RegisterEvent(WebhookNewVoteData{})
}