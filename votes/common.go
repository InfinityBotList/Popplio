package votes

import (
	"context"
	"errors"
	"fmt"
	"popplio/assetmanager"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	entityVoteColsArr = db.GetCols(types.EntityVote{})
	entityVoteCols    = strings.Join(entityVoteColsArr, ",")
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

type EntityInfo struct {
	Name    string
	URL     string
	VoteURL string
	Avatar  string
}

// GetEntityInfo returns information about the entity that is being voted for including vote bans etc.
func GetEntityInfo(ctx context.Context, c DbConn, targetId, targetType string) (*EntityInfo, error) {
	// Handle entity specific checks here, such as ensuring the entity actually exists
	switch targetType {
	case "bot":
		var botType string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT type, vote_banned FROM bots WHERE bot_id = $1", targetId).Scan(&botType, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bot not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch bot data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("bot is vote banned and cannot be voted for right now")
		}

		if botType != "approved" && botType != "certified" {
			return nil, errors.New("bot is not approved or certified and cannot be voted for right now")
		}

		botObj, err := dovewing.GetUser(ctx, targetId, state.DovewingPlatformDiscord)

		if err != nil {
			return nil, err
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/bot/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/bot/" + targetId + "/vote",
			Name:    botObj.Username,
			Avatar:  botObj.Avatar,
		}, nil
	case "pack":
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/pack/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/pack/" + targetId,
			Name:    targetId,
			Avatar:  state.Config.Sites.CDN + "/avatars/default.webp",
		}, nil
	case "team":
		var name string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT name, vote_banned FROM teams WHERE id = $1", targetId).Scan(&name, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("team not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch team data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("team is vote banned and cannot be voted for right now")
		}

		avatar := assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, targetId)

		var avatarPath string

		if avatar.Exists {
			avatarPath = state.Config.Sites.CDN + "/" + avatar.Path + "?ts=" + strconv.FormatInt(avatar.LastModified.Unix(), 10)
		} else {
			avatarPath = state.Config.Sites.CDN + "/" + avatar.DefaultPath
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/team/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/team/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatarPath,
		}, nil
	case "server":
		var name, avatar string
		var voteBanned bool

		err := c.QueryRow(ctx, "SELECT name, avatar, vote_banned FROM servers WHERE server_id = $1", targetId).Scan(&name, &avatar, &voteBanned)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("server not found")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch server data for this vote: %w", err)
		}

		if voteBanned {
			return nil, errors.New("server is vote banned and cannot be voted for right now")
		}

		// Set entityInfo for log
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/server/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/server/" + targetId + "/vote",
			Name:    name,
			Avatar:  avatar,
		}, nil
	case "blog":
		return &EntityInfo{
			URL:     state.Config.Sites.Frontend.Parse() + "/blog/" + targetId,
			VoteURL: state.Config.Sites.Frontend.Parse() + "/blog/" + targetId,
			Name:    targetId,
			Avatar:  state.Config.Sites.CDN + "/avatars/default.webp",
		}, nil
	default:
		return nil, errors.New("unimplemented target type:" + targetType)
	}
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
		"SELECT "+entityVoteCols+" FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3 AND void = false ORDER BY created_at DESC",
		userId,
		targetId,
		targetType,
	)

	if err != nil {
		return nil, err
	}

	validVotes, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.EntityVote])

	if errors.Is(err, pgx.ErrNoRows) {
		validVotes = []types.EntityVote{}
	} else if err != nil {
		return nil, err
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

	err = c.QueryRow(ctx, "SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false AND upvote = false", targetId, targetType).Scan(&downvotes)

	if err != nil {
		return 0, err
	}

	return upvotes - downvotes, nil
}

// Helper function to give votes to an entity based on vote info
func EntityGiveVotes(ctx context.Context, c DbConn, upvote bool, author, targetType, targetId string, vi *types.VoteInfo) error {
	// Keep adding votes until, but not including vi.VoteInfo.PerUser
	for i := 0; i < vi.PerUser; i++ {
		_, err := c.Exec(ctx, "INSERT INTO entity_votes (author, target_id, target_type, upvote, vote_num) VALUES ($1, $2, $3, $4, $5)", author, targetId, targetType, upvote, i)

		if err != nil {
			return fmt.Errorf("failed to insert vote: %w", err)
		}
	}
	return nil
}

// Helper function to perform post-vote tasks
func EntityPostVote(ctx context.Context, c DbConn, author, targetType, targetId string) error {
	nvc, err := EntityGetVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("failed to get vote count: %w", err)
	}

	// Set the approximate vote count
	switch targetType {
	case "bot":
		_, err = c.Exec(ctx, "UPDATE bots SET approximate_votes = $1 WHERE bot_id = $2", nvc, targetId)
	case "server":
		_, err = c.Exec(ctx, "UPDATE servers SET approximate_votes = $1 WHERE server_id = $2", nvc, targetId)
	}

	if err != nil {
		return fmt.Errorf("failed to update vote count: %w", err)
	}

	return nil
}
