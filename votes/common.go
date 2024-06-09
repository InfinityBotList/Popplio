package votes

import (
	"context"
	"popplio/types"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DbConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func GetDoubleVote() bool {
	weekday := time.Now().Weekday()
	return weekday == time.Friday || weekday == time.Saturday || weekday == time.Sunday
}

// Returns core vote info about the entity (such as the amount of cooldown time the entity has)
//
// # If user id is specified, then in the future special perks for the user will be returned as well
//
// If vote time is negative, then it is not possible to revote
func EntityVoteInfo(ctx context.Context, c DbConn, userId, targetId, targetType string) (*types.VoteInfo, error) {
	var voteEntity = types.VoteInfo{
		PerUser:           1,     // 1 vote per user
		VoteTime:          12,    // per day
		MultipleVotes:     true,  // Multiple votes per time interval
		VoteCredits:       false, // Vote credits are not supported unless opted in
		SupportsUpvotes:   true,  // Upvotes are supported (usually)
		SupportsDownvotes: true,  // Downvotes are supported (usually)
	}

	// Add other special cases of entities not following the basic voting system rules
	switch targetType {
	case "bot":
		voteEntity.VoteCredits = true        // Bots support vote credits
		voteEntity.SupportsDownvotes = false // Bots cannot be downvoted

		var premium bool
		err := c.QueryRow(ctx, "SELECT premium FROM bots WHERE bot_id = $1", targetId).Scan(&premium)

		if err != nil {
			return nil, err
		}

		// Premium bots get vote time of 4
		if premium {
			voteEntity.VoteTime = 4
		} else {
			// Bot is not premium
			if GetDoubleVote() {
				voteEntity.PerUser = 2  // 2 votes per user
				voteEntity.VoteTime = 6 // Half of the normal vote time
			}
		}
	case "server":
		voteEntity.VoteCredits = true

		var premium bool
		err := c.QueryRow(ctx, "SELECT premium FROM servers WHERE server_id = $1", targetId).Scan(&premium)

		if err != nil {
			return nil, err
		}

		// Premium servers get vote time of 4
		if premium {
			voteEntity.VoteTime = 4
		} else {
			// Server is not premium
			if GetDoubleVote() {
				voteEntity.PerUser = 2  // 2 votes per user
				voteEntity.VoteTime = 6 // Half of the normal vote time
			}
		}
	case "blog":
		voteEntity.MultipleVotes = false
		voteEntity.PerUser = 1 // Only 1 vote per blog post
	case "team":
		// Teams cannot be premium yet
		if GetDoubleVote() {
			voteEntity.PerUser = 2  // 2 votes per user
			voteEntity.VoteTime = 6 // Half of the normal vote time
		}
	case "pack":
		// Packs cannot be premium yet
		if GetDoubleVote() {
			voteEntity.PerUser = 2  // 2 votes per user
			voteEntity.VoteTime = 6 // Half of the normal vote time
		}
	}

	return &voteEntity, nil
}

// Checks whether or not a user has voted for an entity
func EntityVoteCheck(ctx context.Context, c DbConn, userId, targetId, targetType string) (*types.UserVote, error) {
	vi, err := EntityVoteInfo(ctx, c, userId, targetId, targetType)

	if err != nil {
		return nil, err
	}

	var rows pgx.Rows

	rows, err = c.Query(
		ctx,
		"SELECT created_at, upvote FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3 AND void = false ORDER BY created_at DESC",
		userId,
		targetId,
		targetType,
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

	// If there is a valid vote in this period and the entity supports multiple votes, figure out how long the user has to wait
	var hasVoted bool

	// Case 1: Multiple votes
	if vi.MultipleVotes {
		if len(validVotes) > 0 {
			// Check if the user has voted in the last vote time
			hasVoted = validVotes[0].CreatedAt.Add(time.Duration(vi.VoteTime) * time.Hour).After(time.Now())

			if hasVoted {
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
		}
	} else {
		// Case 2: Single vote entity
		hasVoted = len(validVotes) > 0
	}

	return &types.UserVote{
		HasVoted:   hasVoted,
		ValidVotes: validVotes,
		VoteInfo:   vi,
		Wait:       vw,
	}, nil
}

// Returns the exact (non-cached/approximate) vote count for an entity
func EntityGetVoteCount(ctx context.Context, c DbConn, targetId, targetType string) (int, error) {
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
