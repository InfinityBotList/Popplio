package invite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
)

type ErrorKind string

const (
	ErrGeneric                          ErrorKind = "Generic"
	ErrServerNotFound                   ErrorKind = "ServerNotFound"
	ErrServerNeedsLoginForInvite        ErrorKind = "ServerNeedsLoginForInvite"
	ErrUserIsBlacklisted                ErrorKind = "UserIsBlacklisted"
	ErrServerHasNoInvite                ErrorKind = "ServerHasNoInvite"
	ErrServerHasInvalidInvite           ErrorKind = "ServerHasInvalidInvite"
	ErrServerTypeNotApprovedOrCertified ErrorKind = "ServerTypeNotApprovedOrCertified"
	ErrServerStateNotPublic             ErrorKind = "ServerStateNotPublic"
)

type CreateInviteError struct {
	Kind    ErrorKind
	Message string
}

func (e *CreateInviteError) Error() string {
	switch e.Kind {
	case ErrGeneric:
		return e.Message
	case ErrServerNotFound:
		return "Server not found"
	case ErrServerNeedsLoginForInvite:
		return "In order to view this server, you must login!"
	case ErrUserIsBlacklisted:
		return "User is blacklisted from this server"
	case ErrServerHasNoInvite:
		return "Server has no invite"
	case ErrServerHasInvalidInvite:
		return "Server has an invalid invite"
	case ErrServerTypeNotApprovedOrCertified:
		return "Server is not approved or certified"
	case ErrServerStateNotPublic:
		return "Server is not public/unlisted and hence invites to it cannot be created unless explicitly whitelisted"
	default:
		return string(e.Kind)
	}
}

func generic(format string, args ...any) *CreateInviteError {
	return &CreateInviteError{Kind: ErrGeneric, Message: fmt.Sprintf(format, args...)}
}

func ResolveInvite(ctx context.Context, guildID snowflake.ID, rawInvite string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawInvite, nil)

	if err != nil {
		return err
	}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	url := resp.Request.URL.String()

	if !strings.HasPrefix(url, "https://discord.com/invite/") && !strings.HasPrefix(url, "https://discord.gg") {
		return errors.New("Invalid invite URL")
	}

	parts := strings.Split(url, "/")
	code := parts[len(parts)-1]

	if code == "" {
		return errors.New("Invalid invite URL: No code could be parsed")
	}

	inv, err := dclient.Get().Rest().GetInvite(code)

	if err != nil {
		return fmt.Errorf("Failed to fetch invite: %w", err)
	}

	if inv.Guild == nil {
		return errors.New("Could not fetch information about this guild")
	}

	if inv.Guild.ID != guildID {
		return errors.New("This invite does not correspond to this server")
	}

	if inv.ExpiresAt != nil {
		length := time.Until(*inv.ExpiresAt)

		if length < 30*24*time.Hour {
			return errors.New("Invite expiry must be after at least 30 days long")
		}

		return errors.New("Invite must be permanent")
	}

	return nil
}

func CreateInviteForUser(ctx context.Context, guildID snowflake.ID, userID *string, skipChecks bool) (string, *CreateInviteError) {
	var (
		loginRequired    bool
		blacklistedUsers []string
		inviteStr        string
		serverType       string
		serverState      string
	)

	err := state.Pool.QueryRow(ctx,
		"SELECT login_required_for_invite, blacklisted_users, invite, type, state FROM servers WHERE server_id = $1",
		guildID.String(),
	).Scan(&loginRequired, &blacklistedUsers, &inviteStr, &serverType, &serverState)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", &CreateInviteError{Kind: ErrServerNotFound}
	}

	if err != nil {
		return "", generic("Failed to fetch server data: %v", err)
	}

	if !skipChecks {
		if loginRequired {
			if userID == nil {
				return "", &CreateInviteError{Kind: ErrServerNeedsLoginForInvite}
			}

			if slices.Contains(blacklistedUsers, *userID) {
				return "", &CreateInviteError{Kind: ErrUserIsBlacklisted}
			}
		}

		if serverType != "approved" && serverType != "certified" {
			return "", &CreateInviteError{Kind: ErrServerTypeNotApprovedOrCertified}
		}

		if serverState != "public" && serverState != "unlisted" {
			return "", &CreateInviteError{Kind: ErrServerStateNotPublic}
		}
	}

	if inviteStr == "none" {
		return "", &CreateInviteError{Kind: ErrServerHasNoInvite}
	}

	parts := strings.Split(inviteStr, ":")

	if len(parts) < 2 {
		return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
	}

	switch parts[0] {
	case "invite_url":
		return strings.Join(parts[1:], ":"), nil

	case "per_user":
		channelID, err := snowflake.Parse(parts[1])

		if err != nil {
			return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
		}

		maxUses := 1

		if len(parts) >= 3 {
			n, err := strconv.Atoi(parts[2])

			if err != nil {
				return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
			}

			maxUses = n
		}

		maxAge := 300

		if len(parts) >= 4 {
			n, err := strconv.Atoi(parts[3])

			if err != nil {
				return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
			}

			maxAge = n
		}

		reason := "Invite created for anonymous user"

		if userID != nil {
			reason = fmt.Sprintf("Invite created for user %s", *userID)
		}

		created, err := dclient.Get().Rest().CreateInvite(channelID, discord.InviteCreate{
			MaxAge:  &maxAge,
			MaxUses: &maxUses,
			Unique:  true,
		}, rest.WithReason(reason))

		if err != nil {
			return "", generic("Failed to create invite: %v", err)
		}

		return created.URL(), nil

	default:
		return "", &CreateInviteError{Kind: ErrServerHasInvalidInvite}
	}
}
