package votes

import (
	"context"
	"errors"
	"fmt"
	"popplio/db"
	"popplio/types"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	voteCreditTiersColsArr = db.GetCols(types.VoteCreditTier{})
	voteCreditTiersCols    = strings.Join(voteCreditTiersColsArr, ",")

	entityVoteRedeemLogsColsArr = db.GetCols(types.EntityVoteRedeemLog{})
	entityVoteRedeemLogsCols    = strings.Join(entityVoteRedeemLogsColsArr, ",")
)

// Returns a summary of the vote credit tiers of an entity
func EntityGetVoteCreditsSummary(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
) (*types.VoteCreditTierRedeemSummary, error) {
	rows, err := c.Query(ctx, "SELECT "+voteCreditTiersCols+" FROM vote_credit_tiers WHERE target_type = $1 ORDER BY position ASC", targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote credit tiers [row]: %w", err)
	}

	vcts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.VoteCreditTier])

	if errors.Is(err, pgx.ErrNoRows) {
		vcts = []*types.VoteCreditTier{}
	}

	voteCount, err := EntityGetVoteCount(ctx, c, targetId, targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote count: %w", err)
	}

	slabOverview := SlabSplitVotes(voteCount, vcts)
	totalCredits := SlabCalculateCredits(vcts, slabOverview)

	return &types.VoteCreditTierRedeemSummary{
		Tiers:        vcts,
		Votes:        voteCount,
		SlabOverview: slabOverview,
		TotalCredits: totalCredits,
	}, nil
}

// Redeems vote credits for a user towards a specific entity
func EntityRedeemVoteCredits(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
) error {
	summary, err := EntityGetVoteCreditsSummary(ctx, c, targetId, targetType)

	if err != nil {
		return fmt.Errorf("could not fetch vote credit tiers: %w", err)
	}

	if summary.TotalCredits == 0 {
		return errors.New("no vote credits to redeem")
	}

	var id pgtype.UUID
	err = c.QueryRow(ctx, "INSERT INTO entity_vote_redeem_logs (target_id, target_id, credits) VALUES ($1, $2, $3) RETURNING id", targetId, targetType, summary.TotalCredits).Scan(&id)

	if err != nil {
		return fmt.Errorf("could not log vote credit redemption: %w", err)
	}

	_, err = c.Exec(ctx, "UPDATE entity_votes SET credit_redeem = $1, void = true, void_reason = 'Vote credits redeemed' WHERE target_id = $2 AND target_type = $3 AND void = false", id, targetId, targetType)

	if err != nil {
		return fmt.Errorf("could not redeem vote credits: %w", err)
	}

	return nil
}

// Returns a summary of the entity vote redeem logs
func EntityGetVoteRedeemLogsSummary(
	ctx context.Context,
	c DbConn,
	targetId string,
	targetType string,
) (*types.EntityVoteRedeemLogSummary, error) {
	rows, err := c.Query(ctx, "SELECT "+entityVoteRedeemLogsCols+" FROM entity_vote_redeem_logs WHERE target_id = $1 AND target_type = $2 ORDER BY created_at DESC", targetId, targetType)

	if err != nil {
		return nil, fmt.Errorf("could not fetch vote redeem logs [db fetch]: %w", err)
	}

	evrls, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[types.EntityVoteRedeemLog])

	if errors.Is(err, pgx.ErrNoRows) {
		evrls = []*types.EntityVoteRedeemLog{}
	}

	var totalCredits int
	var redeemedCredits int

	for i := range evrls {
		totalCredits += evrls[i].Credits
		redeemedCredits += evrls[i].RedeemedCredits
	}

	return &types.EntityVoteRedeemLogSummary{
		Redeems:          evrls,
		TotalCredits:     totalCredits,
		RedeemedCredits:  redeemedCredits,
		AvailableCredits: max(totalCredits-redeemedCredits, 0),
	}, nil
}

// Given a number of votes and the vote credit tiers, return the structure of how vote credits should be awarded
// as a map of string to int
//
// Note that this function assumes that the vote credits tiers are sorted by position in ascending order
func SlabSplitVotes(votes int, tiers []*types.VoteCreditTier) []int {
	/*
		<div class="system">
				<p>
					Vote credits are tier based through slabs<br /><br />

					(e.g.)For the following tiers<br /><br />
				</p>
				<OrderedList>
					<ListItem>Tier 1: 100 votes at 0.10 cents</ListItem>
					<ListItem>Tier 2: 200 votes at 0.05 cents</ListItem>
					<ListItem>Tier 3: 50 votes at 0.025 cents</ListItem>
				</OrderedList>
				<p>Would mean 625 votes would be split as the following:</p>
				<OrderedList>
					<ListItem>100 votes: 0.10 cents [Tier 1]</ListItem>
					<ListItem>Next 200 votes: 0.05 cents [Tier 2]</ListItem>
					<ListItem>Next 50 votes: 0.025 cents [Tier 3]</ListItem>
					<ListItem>Last 275 votes: 0.025 cents [last tier used at end of tiering]</ListItem>
				</OrderedList>
			</div>
	*/

	voteCredits := make([]int, len(tiers))

	var remainingVotes = votes

	for i := range tiers {
		if remainingVotes <= 0 {
			break
		}

		if remainingVotes >= tiers[i].Votes {
			voteCredits[i] = tiers[i].Votes
			remainingVotes -= tiers[i].Votes
		} else {
			voteCredits[i] = remainingVotes
			remainingVotes = 0
			break
		}
	}

	// If there are remaining votes, then add them to the last tier
	if remainingVotes > 0 {
		voteCredits[len(tiers)-1] += remainingVotes
	}

	return voteCredits
}

func SlabCalculateCredits(tiers []*types.VoteCreditTier, slab []int) int {
	var totalCredits int

	for i := range tiers {
		totalCredits += tiers[i].Cents * slab[i]
	}

	return totalCredits
}
