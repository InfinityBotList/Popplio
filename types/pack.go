package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// @ci table=packs, unfilled=1
//
// Represents a Bot Pack
type BotPack struct {
	Owner         string                  `db:"owner" json:"-" description:"The owner of the pack"`
	ResolvedOwner *dovetypes.PlatformUser `db:"-" json:"owner" ci:"internal" description:"The resolved owner of the pack"` // Owner must be resolved internally from the owner field
	Name          string                  `db:"name" json:"name" description:"The pack's name"`
	Short         string                  `db:"short" json:"short" description:"The pack's short description"`
	Votes         int                     `db:"-" json:"votes" description:"The pack's vote count" ci:"internal"` // Votes are retrieved from entity_votes
	Tags          []string                `db:"tags" json:"tags" description:"The pack's tags"`
	URL           string                  `db:"url" json:"url" description:"The pack's URL"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at" description:"The pack's creation date"`
	Bots          []string                `db:"bots" json:"bot_ids" description:"The pack's bot IDs"`
	ResolvedBots  []IndexBot              `db:"-" json:"bots" ci:"internal" description:"The resolved bots in the pack"` // Bots must be resolved internally from their IDs
	VoteBanned    bool                    `db:"vote_banned" json:"vote_banned" description:"Whether the pack is banned from voting"`
}
