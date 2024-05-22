package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// Represents a Bot Pack
type BotPack struct {
	Owner         string                  `db:"owner" json:"-" ci:"internal"`
	ResolvedOwner *dovetypes.PlatformUser `db:"-" json:"owner" ci:"internal" description:"The resolved owner of the pack"`
	Name          string                  `db:"name" json:"name" description:"The pack's name"`
	Short         string                  `db:"short" json:"short" description:"The pack's short description"`
	Votes         int                     `db:"-" json:"votes" description:"The pack's vote count" ci:"internal"` // Votes are retrieved from entity_votes
	Tags          []string                `db:"tags" json:"tags" description:"The pack's tags"`
	URL           string                  `db:"url" json:"url" description:"The pack's URL"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at" description:"The pack's creation date"`
	Bots          []string                `db:"bots" json:"bot_ids" description:"The pack's bot IDs"`
	ResolvedBots  []IndexBot              `db:"-" json:"bots" ci:"internal" description:"The resolved bots in the pack"`
}
