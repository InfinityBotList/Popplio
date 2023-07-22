package assets

import (
	"popplio/types"
	"popplio/webhooks/events"
)

const (
	VariableVotes = "$votes"
	VariableUser  = "$user"
)

func changesetOf(t types.WebhookType) types.WebhookType {
	return types.WebhookType(string(types.WebhookTypeChangeset) + "/" + string(t))
}

func GetTestMeta(targetId, targetType string) *types.GetTestWebhookMeta {
	switch targetType {
	case "bot":
		return &types.GetTestWebhookMeta{
			Types: []types.TestWebhookType{
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
							Type: changesetOf(types.WebhookTypeText),
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
							Type: changesetOf(types.WebhookTypeText),
						},
						{
							ID:   "avatar",
							Name: "Avatar",
							Type: changesetOf(types.WebhookTypeText),
						},
					},
				},
			},
		}
	}

	return nil
}
