package votes

import (
	"context"
	"popplio/state"
	"popplio/types"
	"time"
)

// Returns the votes of a pack given the pack's URL/slug
func GetPackVoteData(ctx context.Context, url string) ([]types.PackVote, error) {
	rows, err := state.Pool.Query(ctx, "SELECT user_id, upvote, created_at FROM pack_votes WHERE url = $1", url)

	if err != nil {
		return []types.PackVote{}, err
	}

	defer rows.Close()

	votes := []types.PackVote{}

	for rows.Next() {
		// Fetch votes for the pack
		var userId string
		var upvote bool
		var createdAt time.Time

		err := rows.Scan(&userId, &upvote, &createdAt)

		if err != nil {
			return nil, err
		}

		votes = append(votes, types.PackVote{
			UserID:    userId,
			Upvote:    upvote,
			CreatedAt: createdAt,
		})
	}

	return votes, nil
}
