package seo

import (
	"context"
	"time"
)

type Entity struct {
	// The ID of the entity
	ID string
	// The type of the entity
	Type string
	// Avatar URL
	AvatarURL string
	// The name of the entity
	Name string
	// The description of the entity
	Description string
	// The URL of the entity
	URL string
	// The author of the entity
	Author *Entity
	// When the entity was created
	CreatedAt time.Time
	// When the entity was last updated
	UpdatedAt time.Time
}

type Fetcher interface {
	// The type of the entity
	Type() string
	// Fetches the entity from the database
	Fetch(ctx context.Context, mg *MapGenerator, id string) (*Entity, error)
}

// Generate sitemap/rss feeds easily
type MapGenerator struct {
	Done map[string]map[string]*Entity
}

func (m *MapGenerator) Cache(e *Entity) {
	if m.Done == nil {
		m.Done = make(map[string]map[string]*Entity)
	}

	if m.Done[e.Type] == nil {
		m.Done[e.Type] = make(map[string]*Entity)
	}

	m.Done[e.Type][e.ID] = e
}

func (m *MapGenerator) Add(ctx context.Context, f Fetcher, id string) (*Entity, error) {
	if m.Done == nil {
		m.Done = make(map[string]map[string]*Entity)
	}

	if m.Done[f.Type()] == nil {
		m.Done[f.Type()] = make(map[string]*Entity)
	}

	if m.Done[f.Type()][id] != nil {
		return m.Done[f.Type()][id], nil
	}

	e, err := f.Fetch(ctx, m, id)

	if err != nil {
		return nil, err
	}

	m.Cache(e)

	return e, nil
}
