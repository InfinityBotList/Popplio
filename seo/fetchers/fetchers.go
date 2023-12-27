package fetchers

import (
	"context"
	"fmt"
	"popplio/assetmanager"
	"popplio/seo"
	"popplio/state"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

// Fetcher for a team
type TeamFetcher struct{}

func (t *TeamFetcher) Type() string {
	return "team"
}

func (t *TeamFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	var name string
	var short pgtype.Text
	var createdAt time.Time
	var updatedAt time.Time

	err := state.Pool.QueryRow(ctx, "SELECT name, short, created_at, updated_at FROM teams WHERE id = $1", id).Scan(&name, &short, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	a := assetmanager.AvatarInfo(assetmanager.AssetTargetTypeTeams, id)

	return &seo.Entity{
		ID:        id,
		Type:      t.Type(),
		Name:      name,
		AvatarURL: assetmanager.ResolveAssetMetadataToUrl(a),
		Description: func() string {
			if short.Valid {
				return short.String
			}

			return "This team seems to be a bit mysterious indeed!"
		}(),
		URL:       fmt.Sprintf("%s/teams/%s", state.Config.Sites.Frontend.Production(), id),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// Fetcher for a user
type UserFetcher struct{}

func (u *UserFetcher) Type() string {
	return "user"
}

func (u *UserFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	var count int

	err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE user_id = $1", id).Scan(&count)

	if err != nil {
		return nil, err
	}

	pu, err := dovewing.GetUser(ctx, id, state.DovewingPlatformDiscord)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if count == 0 {
		return &seo.Entity{
			ID:          id,
			Type:        u.Type(),
			AvatarURL:   pu.Avatar,
			Name:        pu.Username,
			Description: "This user seems to be on a distant island somewhere???",
			URL:         fmt.Sprintf("%s/users/%s", state.Config.Sites.Frontend.Production(), id),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	var about string
	var createdAt time.Time
	var updatedAt time.Time

	err = state.Pool.QueryRow(ctx, "SELECT about, created_at, updated_at FROM users WHERE user_id = $1", id).Scan(&about, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	return &seo.Entity{
		ID:          id,
		Type:        u.Type(),
		AvatarURL:   pu.Avatar,
		Name:        pu.Username,
		Description: about,
		URL:         fmt.Sprintf("%s/users/%s", state.Config.Sites.Frontend.Production(), id),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// Fetcher for a bot
type BotFetcher struct{}

func (b *BotFetcher) Type() string {
	return "bot"
}

func (b *BotFetcher) Fetch(ctx context.Context, mg *seo.MapGenerator, id string) (*seo.Entity, error) {
	var short string
	var owner pgtype.Text
	var teamOwner pgtype.Text
	var createdAt time.Time
	var updatedAt time.Time

	err := state.Pool.QueryRow(ctx, "SELECT short, owner, team_owner, created_at, updated_at FROM bots WHERE bot_id = $1", id).Scan(&short, &owner, &teamOwner, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	botUser, err := dovewing.GetUser(ctx, id, state.DovewingPlatformDiscord)

	if err != nil {
		return nil, fmt.Errorf("failed to get bot user: %w", err)
	}

	var resolvedOwner *seo.Entity

	if teamOwner.Valid {
		resolvedOwner, err = mg.Add(ctx, &TeamFetcher{}, teamOwner.String)

		if err != nil {
			return nil, fmt.Errorf("failed to resolve team owner: %w", err)
		}
	}

	if owner.Valid {
		resolvedOwner, err = mg.Add(ctx, &UserFetcher{}, owner.String)

		if err != nil {
			return nil, fmt.Errorf("failed to resolve owner: %w", err)
		}
	}

	return &seo.Entity{
		ID:          id,
		Type:        b.Type(),
		Name:        botUser.Username,
		AvatarURL:   botUser.Avatar,
		Description: short,
		URL:         fmt.Sprintf("%s/bots/%s", state.Config.Sites.Frontend.Production(), id),
		Author:      resolvedOwner,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
