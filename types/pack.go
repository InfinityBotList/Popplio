package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// Represents a Bot Pack
type BotPack struct {
	Owner         string                  `db:"owner" json:"-" ci:"internal"`
	ResolvedOwner *dovetypes.PlatformUser `db:"-" json:"owner"`
	Name          string                  `db:"name" json:"name"`
	Short         string                  `db:"short" json:"short"`
	Votes         int                     `db:"votes" json:"votes" description:"The pack's vote count"`
	Tags          []string                `db:"tags" json:"tags"`
	URL           string                  `db:"url" json:"url"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at"`
	Bots          []string                `db:"bots" json:"bot_ids"`
	ResolvedBots  []IndexBot              `db:"-" json:"bots"`
}

type IndexBotPack struct {
	Owner     string    `db:"owner" json:"owner_id"`
	Name      string    `db:"name" json:"name"`
	Short     string    `db:"short" json:"short"`
	Votes     int       `db:"votes" json:"votes" description:"The pack's vote count"`
	Tags      []string  `db:"tags" json:"tags"`
	URL       string    `db:"url" json:"url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Bots      []string  `db:"bots" json:"bot_ids"`
}
