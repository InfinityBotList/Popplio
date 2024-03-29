package votes

import (
	"context"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/jackc/pgx/v5"
)

func GetDoubleVote() bool {
	return time.Now().Weekday() == time.Friday || time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday
}

func GetVoteTime() uint16 {
	if GetDoubleVote() {
		return 6
	} else {
		return 12
	}
}

// Returns core vote info about the entity (such as the amount of cooldown time the entity has)
//
// If user id is specified, then in the future special perks for the user will be returned as well
func EntityVoteInfo(ctx context.Context, userId, targetId, targetType string) (*types.VoteInfo, error) {
	var defaultVoteEntity = types.VoteInfo{
		PerUser: func() int {
			if GetDoubleVote() {
				return 2
			} else {
				return 1
			}
		}(),
		VoteTime: GetVoteTime(),
	}

	// Add other special cases of entities not following the basic voting system rules
	switch targetType {
	case "bot":
		var premium bool
		err := state.Pool.QueryRow(ctx, "SELECT premium FROM bots WHERE bot_id = $1", targetId).Scan(&premium)

		if err != nil {
			return nil, err
		}

		// Premium bots get vote time of 4
		if premium {
			defaultVoteEntity.VoteTime = 4
		}
	case "server":
		var premium bool
		err := state.Pool.QueryRow(ctx, "SELECT premium FROM servers WHERE server_id = $1", targetId).Scan(&premium)

		if err != nil {
			return nil, err
		}

		// Premium bots get vote time of 4
		if premium {
			defaultVoteEntity.VoteTime = 4
		}
	}

	return &defaultVoteEntity, nil
}

// Checks whether or not a user has voted for an entity
func EntityVoteCheck(ctx context.Context, userId, targetId, targetType string) (*types.UserVote, error) {
	vi, err := EntityVoteInfo(ctx, userId, targetId, targetType)

	if err != nil {
		return nil, err
	}

	rows, err := state.Pool.Query(
		ctx,
		"SELECT created_at, upvote FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3 AND void = false AND NOW() - created_at < make_interval(hours => $4) ORDER BY created_at DESC",
		userId,
		targetId,
		targetType,
		vi.VoteTime,
	)

	if err != nil {
		return nil, err
	}

	var validVotes []*types.ValidVote

	for rows.Next() {
		var createdAt time.Time
		var upvote bool

		err = rows.Scan(&createdAt, &upvote)

		if err != nil {
			return nil, err
		}

		validVotes = append(validVotes, &types.ValidVote{
			Upvote:    upvote,
			CreatedAt: createdAt,
		})
	}

	var vw *types.VoteWait

	if len(validVotes) > 0 {
		timeElapsed := time.Since(validVotes[0].CreatedAt)

		timeToWait := int64(vi.VoteTime)*60*60*1000 - timeElapsed.Milliseconds()

		timeToWaitTime := (time.Duration(timeToWait) * time.Millisecond)

		hours := timeToWaitTime / time.Hour
		mins := (timeToWaitTime - (hours * time.Hour)) / time.Minute
		secs := (timeToWaitTime - (hours*time.Hour + mins*time.Minute)) / time.Second

		vw = &types.VoteWait{
			Hours:   int(hours),
			Minutes: int(mins),
			Seconds: int(secs),
		}
	}

	return &types.UserVote{
		HasVoted:   len(validVotes) > 0,
		ValidVotes: validVotes,
		VoteInfo:   vi,
		Wait:       vw,
	}, nil
}

type GVCConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func EntityGetVoteCount(ctx context.Context, c GVCConn, targetId, targetType string) (int, error) {
	var upvotes int
	var downvotes int

	err := c.QueryRow(ctx, "SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false AND upvote = true", targetId, targetType).Scan(&upvotes)

	if err != nil {
		return 0, err
	}

	// Bots can never have a downvote at this time
	if targetType != "bot" {
		err = c.QueryRow(ctx, "SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false AND upvote = false", targetId, targetType).Scan(&downvotes)

		if err != nil {
			return 0, err
		}
	}

	return upvotes - downvotes, nil
}
