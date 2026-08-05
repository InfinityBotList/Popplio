package panel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// parsePartner is the shared validator for Create and Update.
func parsePartner(ctx context.Context, partner types.CreatePartner) error {
	var typeID string

	err := state.Pool.QueryRow(ctx, "SELECT id FROM partner_types WHERE id = $1", partner.Type).Scan(&typeID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("Partner type does not exist")
		}

		return err
	}

	// Upstream also required the partner avatar to already exist on the CDN and
	// checked its size here. That validation went with the CDN; see
	// CONFORMANCE.md.
	if len(partner.Links) == 0 {
		return errors.New("Links cannot be empty")
	}

	for _, link := range partner.Links {
		if link.Name == "" {
			return errors.New("Link name cannot be empty")
		}

		if link.Value == "" {
			return errors.New("Link URL cannot be empty")
		}

		if !strings.HasPrefix(link.Value, "https://") {
			return errors.New("Link URL must start with https://")
		}
	}

	var userID string

	err = state.Pool.QueryRow(ctx, "SELECT user_id FROM users WHERE user_id = $1", partner.UserID).Scan(&userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("User does not exist")
		}

		return err
	}

	return nil
}

type partnerRow struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Short     string    `db:"short"`
	Links     []byte    `db:"links"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
	UserID    string    `db:"user_id"`
	BotID     *string   `db:"bot_id"`
}

type partnerTypeRow struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Short     string    `db:"short"`
	Icon      string    `db:"icon"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Server) updatePartners(ctx context.Context, q *types.QUpdatePartners) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx, "SELECT id, name, short, links, type, created_at, user_id, bot_id FROM partners")

		if err != nil {
			return response{}, newError(err)
		}

		partnerRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[partnerRow])

		if err != nil {
			return response{}, newError(err)
		}

		partners := make([]types.Partner, 0, len(partnerRows))

		for _, p := range partnerRows {
			var links []types.Link

			if err := json.Unmarshal(p.Links, &links); err != nil {
				return response{}, newError(err)
			}

			partners = append(partners, types.Partner{
				ID:        p.ID,
				Name:      p.Name,
				Short:     p.Short,
				Links:     types.NonNilLinks(links),
				Type:      p.Type,
				CreatedAt: types.NewTimestamp(p.CreatedAt),
				UserID:    p.UserID,
				BotID:     p.BotID,
			})
		}

		rows, err = state.Pool.Query(ctx, "SELECT id, name, short, icon, created_at FROM partner_types")

		if err != nil {
			return response{}, newError(err)
		}

		typeRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[partnerTypeRow])

		if err != nil {
			return response{}, newError(err)
		}

		partnerTypes := make([]types.PartnerType, 0, len(typeRows))

		for _, t := range typeRows {
			partnerTypes = append(partnerTypes, types.PartnerType{
				ID:        t.ID,
				Name:      t.Name,
				Short:     t.Short,
				Icon:      t.Icon,
				CreatedAt: types.NewTimestamp(t.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, types.Partners{Partners: partners, PartnerTypes: partnerTypes}), nil
	case q.Action.Create != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to create partners [manage_partners]"), nil
		}

		partner := q.Action.Create.Partner

		exists, err := partnerExists(ctx, partner.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if exists {
			return writeText(http.StatusBadRequest, "Partner already exists"), nil
		}

		if err := parsePartner(ctx, partner); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		links, err := json.Marshal(partner.Links)

		if err != nil {
			return response{}, newError(err)
		}

		_, err = state.Pool.Exec(ctx,
			"INSERT INTO partners (id, name, short, links, type, user_id, bot_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			partner.ID, partner.Name, partner.Short, links, partner.Type, partner.UserID, partner.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Update != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to update partners [manage_partners]"), nil
		}

		partner := q.Action.Update.Partner

		exists, err := partnerExists(ctx, partner.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Partner does not already exist"), nil
		}

		if err := parsePartner(ctx, partner); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		links, err := json.Marshal(partner.Links)

		if err != nil {
			return response{}, newError(err)
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE partners SET name = $2, short = $3, links = $4, type = $5, user_id = $6, bot_id = $7 WHERE id = $1",
			partner.ID, partner.Name, partner.Short, links, partner.Type, partner.UserID, partner.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManagePartners) {
			return writeText(http.StatusForbidden, "You do not have permission to delete partners [manage_partners]"), nil
		}

		id := q.Action.Delete.ID

		exists, err := partnerExists(ctx, id)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Partner does not exist"), nil
		}

		// Upstream also deleted the partner's CDN image here. That went with the
		// CDN; see CONFORMANCE.md.
		if _, err := state.Pool.Exec(ctx, "DELETE FROM partners WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No partner action was specified")
	}
}

func partnerExists(ctx context.Context, id string) (bool, error) {
	var found string

	err := state.Pool.QueryRow(ctx, "SELECT id FROM partners WHERE id = $1", id).Scan(&found)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// updateChangelog is a hard stub: it always returns 403 regardless of input or
// authentication. The ChangelogAction DTOs still parse, which is why they exist.
func (s *Server) updateChangelog(_ context.Context, _ *types.QUpdateChangelog) (response, error) {
	return writeText(http.StatusForbidden, "You do not have permission to create changelog entries [not implemented]"), nil
}

type blogRow struct {
	Itag        pgtype.UUID `db:"itag"`
	Slug        string      `db:"slug"`
	Title       string      `db:"title"`
	Description string      `db:"description"`
	UserID      string      `db:"user_id"`
	Content     string      `db:"content"`
	CreatedAt   time.Time   `db:"created_at"`
	Draft       bool        `db:"draft"`
	Tags        []string    `db:"tags"`
}

func (s *Server) updateBlog(ctx context.Context, q *types.QUpdateBlog) (response, error) {
	// Upstream calls check_auth twice here with a TODO admitting it is wasteful;
	// once is enough.
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListEntries != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT itag, slug, title, description, user_id, content, created_at, draft, tags FROM blogs ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		blogRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[blogRow])

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.BlogPost, 0, len(blogRows))

		for _, row := range blogRows {
			entries = append(entries, types.BlogPost{
				Itag:        impls.UUIDString(row.Itag),
				Slug:        row.Slug,
				Title:       row.Title,
				Description: row.Description,
				UserID:      row.UserID,
				Tags:        types.NonNilStrings(row.Tags),
				Content:     row.Content,
				CreatedAt:   types.NewTimestamp(row.CreatedAt),
				Draft:       row.Draft,
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.CreateEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to create blog entries [manage_blog]"), nil
		}

		entry := q.Action.CreateEntry

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO blogs (slug, title, description, content, tags, user_id) VALUES ($1, $2, $3, $4, $5, $6)",
			entry.Slug, entry.Title, entry.Description, entry.Content, entry.Tags, authData.UserID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.UpdateEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to update blog entries [manage_blog]"), nil
		}

		entry := q.Action.UpdateEntry

		itag, err := uuid.Parse(entry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		exists, err := blogExists(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry does not exist"), nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE blogs SET slug = $2, title = $3, description = $4, content = $5, tags = $6, draft = $7 WHERE itag = $1",
			itag, entry.Slug, entry.Title, entry.Description, entry.Content, entry.Tags, entry.Draft)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.DeleteEntry != nil:
		if !userPerms.Has(perms.StaffManageBlog) {
			return writeText(http.StatusForbidden, "You do not have permission to delete blog entries [manage_blog]"), nil
		}

		itag, err := uuid.Parse(q.Action.DeleteEntry.Itag)

		if err != nil {
			return response{}, newError(err)
		}

		exists, err := blogExists(ctx, itag)

		if err != nil {
			return response{}, newError(err)
		}

		if !exists {
			return writeText(http.StatusBadRequest, "Entry with same id does not already exist"), nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM blogs WHERE itag = $1", itag); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No blog action was specified")
	}
}

func blogExists(ctx context.Context, itag uuid.UUID) (bool, error) {
	var count int64

	if err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM blogs WHERE itag = $1", itag).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}
