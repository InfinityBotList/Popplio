package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// The panel's way into the shared action layer: running an RPC method, listing
// what methods exist, and reading the log of what has been run.
//
// executeRpc deliberately does no permission check of its own — the rpc package
// enforces the per-method permission, so both callers get the same answer.

func (s *Server) executeRpc(ctx context.Context, q *types.QExecuteRpc) (response, error) {
	// The endpoint is open to all staff; the RPC layer enforces per-method perms.
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	resp, rpcErr := rpc.Execute(ctx, q.Method, rpc.Handle{
		UserID:     authData.UserID,
		TargetType: q.TargetType,
	})

	if rpcErr != nil {
		// RPC failures are 400, not 500.
		return writeText(http.StatusBadRequest, rpcErr.Error()), nil
	}

	if content, ok := resp.Text(); ok {
		return writeText(http.StatusOK, content), nil
	}

	return writeNoContent(), nil
}

func (s *Server) getRpcMethods(ctx context.Context, q *types.QGetRpcMethods) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	actions := make([]types.RPCWebAction, 0, len(types.RPCMethodVariants))

	for _, name := range types.RPCMethodVariants {
		variant, err := types.EmptyRPCMethod(name)

		if err != nil {
			return response{}, newError(err)
		}

		required, known := types.RPCPermission(name)

		if q.Filtered && (!known || !userPerms.Has(required)) {
			continue
		}

		actions = append(actions, types.RPCWebAction{
			ID:                   name,
			Label:                variant.Label(),
			Description:          variant.Description(),
			SupportedTargetTypes: variant.SupportedTargetTypes(),
			Fields:               variant.Fields(),
		})
	}

	return writeJSON(http.StatusOK, actions), nil
}

type rpcLogRow struct {
	ID        pgtype.UUID `db:"id"`
	UserID    string      `db:"user_id"`
	Method    string      `db:"method"`
	Data      []byte      `db:"data"`
	State     string      `db:"state"`
	CreatedAt time.Time   `db:"created_at"`
}

func (s *Server) getRpcLogEntries(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	if !userPerms.Has(perms.StaffViewAuditLogs) {
		return writeText(http.StatusForbidden, "You do not have permission to view rpc logs [view_audit_logs]"), nil
	}

	rows, err := state.Pool.Query(ctx, "SELECT id, user_id, method, data, state, created_at FROM rpc_logs ORDER BY created_at DESC")

	if err != nil {
		return response{}, newError(err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[rpcLogRow])

	if err != nil {
		return response{}, newError(err)
	}

	log := make([]types.RPCLogEntry, 0, len(entries))

	for _, entry := range entries {
		log = append(log, types.RPCLogEntry{
			ID:        impls.UUIDString(entry.ID),
			UserID:    entry.UserID,
			Method:    entry.Method,
			Data:      entry.Data,
			State:     entry.State,
			CreatedAt: types.NewTimestamp(entry.CreatedAt),
		})
	}

	return writeJSON(http.StatusOK, log), nil
}
