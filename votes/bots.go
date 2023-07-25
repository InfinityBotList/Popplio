package votes

import (
	"context"
	"popplio/state"
	"popplio/types"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func GetBotVoteData(ctx context.Context, userID, botID string, log bool) (*types.UserVote, error) {
	var premium bool
	err := state.Pool.QueryRow(ctx, "SELECT premium FROM bots WHERE bot_id = $1", botID).Scan(&premium)

	if err != nil {
		return nil, err
	}

	var votes []int64

	var voteDates []*struct {
		Date pgtype.Timestamptz `db:"created_at"`
	}

	rows, err := state.Pool.Query(ctx, "SELECT created_at FROM votes WHERE user_id = $1 AND bot_id = $2 ORDER BY created_at DESC", userID, botID)

	if err != nil {
		return nil, err
	}

	err = pgxscan.ScanAll(&voteDates, rows)

	for _, vote := range voteDates {
		if vote.Date.Valid {
			votes = append(votes, vote.Date.Time.UnixMilli())
		}
	}

	voteParsed := types.UserVote{
		UserID: userID,
		VoteInfo: types.VoteInfo{
			Weekend:  GetDoubleVote(),
			VoteTime: GetVoteTime(),
		},
		PremiumBot: premium,
	}

	if premium {
		voteParsed.VoteInfo.VoteTime = 4
	}

	if log {
		state.Logger.With(
			zap.String("user_id", userID),
			zap.String("bot_id", botID),
			zap.Int64s("votes", votes),
			zap.Error(err),
		).Info("Got vote data")
	}

	voteParsed.Timestamps = votes

	// In most cases, will be one but not always
	if len(votes) > 0 {
		if time.Now().UnixMilli() < votes[0] {
			state.Logger.Error("detected illegal vote time", votes[0])
			votes[0] = time.Now().UnixMilli()
		}

		if time.Now().UnixMilli()-votes[0] < int64(voteParsed.VoteInfo.VoteTime)*60*60*1000 {
			voteParsed.HasVoted = true
			voteParsed.LastVoteTime = votes[0]
		}
	}

	if voteParsed.LastVoteTime == 0 && len(votes) > 0 {
		voteParsed.LastVoteTime = votes[0]
	}
	return &voteParsed, nil
}
