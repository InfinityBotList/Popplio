// Package rpc is the shared action layer behind both the staff panel and the
// staff Discord bot. A staff member approving a bot from the web panel and one
// approving it with a slash command run the same handler, the same permission
// check, the same rate limit, the same audit row and the same mod-log embed.
//
// # Layout
//
// core.go holds the pipeline every action goes through (Execute) and the guards
// they share; dispatch.go maps a method to its handler; the handlers themselves
// are grouped by what they act on and by the permission that gates them:
//
//	review.go       claim, unclaim, approve, deny, unverify   review_bots
//	certify.go      certification                             certify_bots
//	transfer.go     ownership changes                         transfer_bots
//	forceremove.go  deleting a bot outright                   force_remove_bots
//	premium.go      premium                                   manage_premium
//	votes.go        vote bans and vote resets                 manage_votes, ban_voters
//	apps.go         application bans                          ban_app_users
//	audit.go        the staff_general_logs row
//
// # Embeds are reproduced verbatim
//
// Every mod-log embed in this package — titles (leading spaces included),
// descriptions, field names, inline flags, footers and colours — is reproduced
// exactly from the Rust original. The mod-log channel is read by humans and
// several titles have odd leading spaces that are intentional by accident. See
// arcadia/CONFORMANCE.md before tidying one up.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

// Handle carries the caller identity and target type for one RPC execution.
type Handle struct {
	UserID     string
	TargetType types.TargetType
}

// Success is either an empty result (204 from the panel) or a text payload
// (200 text/plain from the panel).
type Success struct {
	content *string
}

// NoContent is a successful result with no body.
func NoContent() Success { return Success{} }

// Content is a successful result carrying raw text.
func Content(s string) Success { return Success{content: &s} }

// Text returns the payload and whether one was set.
func (s Success) Text() (string, bool) {
	if s.content == nil {
		return "", false
	}

	return *s.content, true
}

// Execute runs the full RPC pipeline: target-type check, permission check, audit
// row, rate limit, handler, then the audit row's final state.
//
// The order matters and is reproduced exactly. In particular the audit row is
// inserted BEFORE the rate limit is counted, so the current call counts towards
// its own limit: the effective allowance is 5 calls per rolling 7 minutes and the
// 6th is rejected.
func Execute(ctx context.Context, method types.RPCMethod, h Handle) (Success, error) {
	if !supportsTargetType(method, h.TargetType) {
		return Success{}, errors.New("This method does not support this target type yet")
	}

	userPerms, err := impls.GetUserPerms(ctx, h.UserID)

	if err != nil {
		return Success{}, err
	}

	required, ok := method.Permission()

	if !ok {
		return Success{}, fmt.Errorf("Unknown method %s", method.Name())
	}

	if !userPerms.Has(required) {
		return Success{}, fmt.Errorf("You need the %s permission to use %s", required, method.Name())
	}

	data, err := json.Marshal(method)

	if err != nil {
		return Success{}, err
	}

	var logID pgtype.UUID

	err = state.Pool.QueryRow(ctx,
		"INSERT INTO rpc_logs (method, user_id, data) VALUES ($1, $2, $3) RETURNING id",
		method.Name(), h.UserID, data,
	).Scan(&logID)

	if err != nil {
		return Success{}, err
	}

	var count int64

	err = state.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM rpc_logs WHERE user_id = $1 AND NOW() - created_at < INTERVAL '7 minutes'",
		h.UserID,
	).Scan(&count)

	if err != nil {
		return Success{}, errors.New("Failed to get ratelimit count")
	}

	if count > 5 {
		_, err = state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE user_id = $1", h.UserID)

		if err != nil {
			return Success{}, errors.New("Failed to reset user token")
		}

		return Success{}, errors.New("Rate limit exceeded. Wait 5-10 minutes and try again?")
	}

	resp, handlerErr := handleMethod(ctx, method, h)

	logState := "success"
	if handlerErr != nil {
		logState = handlerErr.Error()
	}

	if _, err := state.Pool.Exec(ctx, "UPDATE rpc_logs SET state = $1 WHERE id = $2", logState, logID); err != nil {
		return Success{}, err
	}

	return resp, handlerErr
}

func supportsTargetType(method types.RPCMethod, target types.TargetType) bool {
	for _, t := range method.SupportedTargetTypes() {
		if t == target {
			return true
		}
	}

	return false
}

// checkReason enforces MAX_REASON_LENGTH. The message is user-visible.
func checkReason(reason string) error {
	if len(reason) > types.MaxReasonLength {
		return fmt.Errorf("Reason must be lower than/equal to %d characters", types.MaxReasonLength)
	}

	return nil
}

// entityExists is the "SELECT COUNT(*)" existence guard several methods run. The
// error string genuinely starts with a space upstream: the entity name was
// dropped in an earlier refactor and the panel shows the message raw.
func entityExists(ctx context.Context, query string, targetID string) error {
	var count int64

	if err := state.Pool.QueryRow(ctx, query, targetID).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		return errors.New(" does not exist")
	}

	return nil
}

func botExists(ctx context.Context, targetID string) error {
	return entityExists(ctx, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", targetID)
}

func userExists(ctx context.Context, targetID string) error {
	return entityExists(ctx, "SELECT COUNT(*) FROM users WHERE user_id = $1", targetID)
}
