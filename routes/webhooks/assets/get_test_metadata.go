package assets

import (
	"popplio/types"
	"popplio/webhooks/events"
)

const (
	VariableVotes = "$votes"
	VariableUser  = "$user"
)

func GetTestMeta(targetId, targetType string) *types.GetTestWebhookMeta {
	switch targetType {
	case "bot":
		return &types.GetTestWebhookMeta{
			Types: []types.TestWebhookType{
				{
					Type: string(events.WebhookTypeBotVote),
					Data: []types.TestWebhookVariables{
						{
							ID:    "votes",
							Name:  "Number Of Votes",
							Type:  types.WebhookTypeNumber,
							Value: VariableVotes,
						},
					},
				},
				{
					Type: string(events.WebhookTypeBotEditReview),
					Data: []types.TestWebhookVariables{
						{
							ID:   "review_id",
							Name: "Review ID",
							Type: types.WebhookTypeText,
						},
						{
							ID:   "content",
							Name: "Content",
							Type: types.WebhookTypeChangeset,
						},
					},
				},
				{
					Type: string(events.WebhookTypeBotNewReview),
					Data: []types.TestWebhookVariables{
						{
							ID:   "review_id",
							Name: "Review ID",
							Type: types.WebhookTypeText,
						},
						{
							ID:   "content",
							Name: "Content",
							Type: types.WebhookTypeText,
						},
					},
				},
			},
		}
	case "team":
		return &types.GetTestWebhookMeta{
			Types: []types.TestWebhookType{
				{
					Type: string(events.WebhookTypeTeamEdit),
					Data: []types.TestWebhookVariables{
						{
							ID:   "name",
							Name: "Name",
							Type: types.WebhookTypeChangeset,
						},
						{
							ID:   "avatar",
							Name: "Avatar",
							Type: types.WebhookTypeChangeset,
						},
					},
				},
			},
		}
	}

	return nil
}
