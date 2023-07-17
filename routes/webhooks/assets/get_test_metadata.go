package assets

import (
	"popplio/types"
	"popplio/webhooks/events"
)

const (
	VariableVotes = "$votes"
	VariableUser  = "$user"
)

func GetTestMeta(targetId, targetType string, webhooksV2 bool) *types.GetTestWebhookMeta {
	switch targetType {
	case "bot":
		if webhooksV2 {
			return &types.GetTestWebhookMeta{
				Types: []types.TestWebhookType{
					{
						Type: string(events.WebhookTypeBotVote),
						Data: []types.TestWebhookVariables{
							{
								ID:    "votes",
								Name:  "Number Of Votes",
								Type:  "number",
								Value: VariableVotes,
							},
							{
								ID:    "author",
								Name:  "Author ID",
								Type:  "string",
								Value: VariableUser,
							},
						},
					},
					{
						Type: string(events.WebhookTypeBotEditReview),
						Data: []types.TestWebhookVariables{
							{
								ID:    "author",
								Name:  "Author ID",
								Type:  "string",
								Value: VariableUser,
							},
							{
								ID:   "content",
								Name: "Content",
								Type: "changeset",
							},
						},
					},
					{
						Type: string(events.WebhookTypeBotNewReview),
						Data: []types.TestWebhookVariables{
							{
								ID:    "author",
								Name:  "Author ID",
								Type:  "string",
								Value: VariableUser,
							},
							{
								ID:   "content",
								Name: "Content",
								Type: "string",
							},
						},
					},
				},
			}
		} else {
			return &types.GetTestWebhookMeta{
				// Just vote
				Types: []types.TestWebhookType{
					{
						Type: string(events.WebhookTypeBotVote),
						Data: []types.TestWebhookVariables{
							{
								ID:    "votes",
								Name:  "Number Of Votes",
								Type:  "number",
								Value: VariableVotes,
							},
							{
								ID:    "author",
								Name:  "Author ID",
								Type:  "string",
								Value: VariableUser,
							},
						},
					},
				},
			}
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
							Type: "changeset",
						},
						{
							ID:   "avatar",
							Name: "Avatar",
							Type: "changeset",
						},
					},
				},
			},
		}
	}

	return nil
}
