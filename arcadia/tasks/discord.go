package tasks

import (
	"context"
	"fmt"
	"strings"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// BansSync reconciles the main guild's ban list with users.banned.
func BansSync(ctx context.Context) error {
	bans, err := fetchAllBans(state.Config.Servers.Main)

	if err != nil {
		return fmt.Errorf("Error while fetching bans: %s", err)
	}

	rows, err := state.Pool.Query(ctx, "SELECT user_id FROM users WHERE banned = true")

	if err != nil {
		return fmt.Errorf("Error while fetching bans from database: %s", err)
	}

	dbBanned, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return fmt.Errorf("Error while fetching bans from database: %s", err)
	}

	var pingUsers strings.Builder

	for _, owner := range state.Config.Arcadia.Owners {
		fmt.Fprintf(&pingUsers, "<@%s>", owner)
	}

	serverBans := make(map[string]struct{}, len(bans))

	for _, ban := range bans {
		serverBans[ban.User.ID.String()] = struct{}{}
	}

	dbBans := make(map[string]struct{}, len(dbBanned))

	for _, userID := range dbBanned {
		dbBans[userID] = struct{}{}
	}

	// Symmetric difference: in the guild but not the DB means "ban them", in the
	// DB but not the guild means "unban them".
	toModify := make([]string, 0)

	for userID := range serverBans {
		if _, ok := dbBans[userID]; !ok {
			toModify = append(toModify, userID)
		}
	}

	for userID := range dbBans {
		if _, ok := serverBans[userID]; !ok {
			toModify = append(toModify, userID)
		}
	}

	state.Logger.Warn("Bans to modify", zap.Strings("users", toModify))

	for _, userID := range toModify {
		_, isBanned := serverBans[userID]

		tag, err := state.Pool.Exec(ctx, "UPDATE users SET banned = $1 WHERE user_id = $2", isBanned, userID)

		if err != nil {
			return fmt.Errorf("Error while updating user %s in database: %v", userID, err)
		}

		if tag.RowsAffected() == 0 {
			_, err := state.Pool.Exec(ctx,
				"INSERT INTO users (user_id, banned, api_token) VALUES ($1, $2, $3)",
				userID, isBanned, impls.GenRandom(512))

			if err != nil {
				return fmt.Errorf("Error while inserting user %s into database: %v", userID, err)
			}
		}

		title := "User Unban"
		description := fmt.Sprintf("User %s was unbanned", userID)
		colour := impls.ColourBlurple

		if isBanned {
			title = "User Ban"
			description = fmt.Sprintf("User %s was banned", userID)
			colour = impls.ColourRed
		}

		err = impls.SendModLog(discord.MessageCreate{
			Content: pingUsers.String(),
			Embeds: []discord.Embed{{
				Title:       title,
				Description: description,
				Color:       colour,
			}},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// fetchAllBans pages through the guild ban list.
func fetchAllBans(guildID snowflake.ID) ([]discord.Ban, error) {
	var (
		all   []discord.Ban
		after snowflake.ID
	)

	for {
		page, err := dclient.Get().Rest().GetBans(guildID, 0, after, 1000)

		if err != nil {
			return nil, err
		}

		all = append(all, page...)

		if len(page) < 1000 {
			return all, nil
		}

		after = page[len(page)-1].User.ID
	}
}

// SpecRoleSync mirrors the bug hunter role into users.bug_hunters.
func SpecRoleSync(ctx context.Context) error {
	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Main); !ok {
		return fmt.Errorf("Failed to get guild")
	}

	var bugHunters []string

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Main, func(member discord.Member) {
		for _, roleID := range member.RoleIDs {
			if roleID == state.Config.Roles.BugHunters {
				bugHunters = append(bugHunters, member.User.ID.String())
				return
			}
		}
	})

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("Error creating transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE users SET bug_hunters = false"); err != nil {
		return fmt.Errorf("Error updating users: %v", err)
	}

	for _, userID := range bugHunters {
		if _, err := tx.Exec(ctx, "UPDATE users SET bug_hunters = true WHERE user_id = $1", userID); err != nil {
			return fmt.Errorf("Error updating users: %v", err)
		}
	}

	return tx.Commit(ctx)
}

// VoteResetter voids every non-immutable vote once a month.
func VoteResetter(ctx context.Context) error {
	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var id string

	err = tx.QueryRow(ctx, "SELECT id FROM automated_vote_resets WHERE created_at > NOW() - INTERVAL '1 month' FOR UPDATE").Scan(&id)

	if err == nil {
		// A reset already happened this month.
		return nil
	}

	if !isNoRows(err) {
		return err
	}

	if _, err := tx.Exec(ctx, "LOCK TABLE entity_votes IN EXCLUSIVE MODE"); err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		"UPDATE entity_votes SET void = TRUE, void_reason = 'Automated votes reset', voided_at = NOW() WHERE void = false AND immutable = false")

	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "INSERT INTO automated_vote_resets (created_at) VALUES (NOW())"); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return impls.SendModLog(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:  "__Automated Per-Monthly Vote Reset!__",
			Footer: impls.Footer("Welcome back :)"),
			Color:  impls.ColourRed,
		}},
	})
}

func isNoRows(err error) bool {
	return err != nil && err.Error() == pgx.ErrNoRows.Error()
}

// topReviewerQuery ranks reviewers by their Approve+Deny count.
const topReviewerQuery = "SELECT user_id, approved_count, denied_count, total_count FROM (SELECT rpc.user_id, SUM(CASE WHEN rpc.method = 'Approve' THEN 1 ELSE 0 END) AS approved_count, SUM(CASE WHEN rpc.method = 'Deny' THEN 1 ELSE 0 END) AS denied_count, SUM(CASE WHEN rpc.method IN ('Approve', 'Deny') THEN 1 ELSE 0 END) AS total_count FROM rpc_logs rpc LEFT JOIN staff_members sm ON rpc.user_id = sm.user_id WHERE rpc.method IN ('Approve', 'Deny') AND sm.user_id IS NOT NULL GROUP BY rpc.user_id) AS subquery WHERE total_count > 0 ORDER BY total_count DESC LIMIT $1"

// TopReviewer is one row of the leaderboard query.
type TopReviewer struct {
	UserID        string `db:"user_id"`
	ApprovedCount int64  `db:"approved_count"`
	DeniedCount   int64  `db:"denied_count"`
	TotalCount    int64  `db:"total_count"`
}

// QueryTopReviewers runs the leaderboard query. The Discord `leaderboard` and
// `refresh` commands share it.
func QueryTopReviewers(ctx context.Context, limit int) ([]TopReviewer, error) {
	rows, err := state.Pool.Query(ctx, topReviewerQuery, limit)

	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[TopReviewer])
}

// TopReviewerSync strips the top-reviewer role from everyone in the main guild
// and re-grants it to the top reviewers.
//
// BUG (reproduced): the limit is hardcoded to 0, so the query selects NOBODY and
// this weekly job only ever strips the role. The Discord `refresh` command uses
// a limit of 3. Left as-is by explicit decision; see CONFORMANCE.md.
func TopReviewerSync(ctx context.Context) error {
	const limit = 0

	stats, err := QueryTopReviewers(ctx, limit)

	if err != nil {
		return err
	}

	if _, ok := dclient.Get().Caches().Guild(state.Config.Servers.Main); !ok {
		state.Logger.Error("Failed to get guild")
		return nil
	}

	if err := SyncTopReviewerRoles(ctx, stats); err != nil {
		return err
	}

	return impls.SendModLog(discord.MessageCreate{
		Content: "**Weekly Job**\nSynced Top Reviewers!",
	})
}

// SyncTopReviewerRoles removes the top-reviewer role from every cached main
// guild member and then grants it to the given reviewers. Shared with the
// Discord `refresh` command.
func SyncTopReviewerRoles(ctx context.Context, stats []TopReviewer) error {
	const reason = "Syncing top reviewers, weekly job."

	var holders []snowflake.ID

	dclient.Get().Caches().MembersForEach(state.Config.Servers.Main, func(member discord.Member) {
		for _, roleID := range member.RoleIDs {
			if roleID == state.Config.Roles.TopReviewers {
				holders = append(holders, member.User.ID)
				return
			}
		}
	})

	for _, userID := range holders {
		if err := impls.RemoveRole(state.Config.Servers.Main, userID, state.Config.Roles.TopReviewers, reason); err != nil {
			state.Logger.Error("Failed to remove role from member", zap.String("userID", userID.String()), zap.Error(err))
		}
	}

	for _, stat := range stats {
		userID, err := snowflake.Parse(stat.UserID)

		if err != nil {
			state.Logger.Error("Failed to parse user_id", zap.String("userID", stat.UserID))
			continue
		}

		if !impls.MemberOnGuild(state.Config.Servers.Main, userID) {
			continue
		}

		if err := impls.AddRole(state.Config.Servers.Main, userID, state.Config.Roles.TopReviewers, reason); err != nil {
			state.Logger.Error("Failed to add role to user", zap.String("userID", stat.UserID), zap.Error(err))
		}
	}

	return nil
}
