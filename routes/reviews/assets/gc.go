package assets

import (
	"context"
	"popplio/state"
	"popplio/types"
)

// The GC step is needed to kill any reviews whose parent has been deleted etc.
func GarbageCollect(ctx context.Context, reviews []types.Review) ([]types.Review, error) {
	var okReviews []types.Review = []types.Review{}
	var hasDeleted bool
	for i := range reviews {
		// Case 1: The review has no parent
		if !reviews[i].ParentID.Valid {
			okReviews = append(okReviews, reviews[i])
			continue
		}
		// Case 2: The review has a parent
		var found bool = false
		for j := range reviews {
			if reviews[i].ParentID.Bytes == reviews[j].ID.Bytes {
				found = true
				break
			}
		}

		if found {
			okReviews = append(okReviews, reviews[i])
		} else {
			// Delete the review
			_, err := state.Pool.Exec(ctx, "DELETE FROM reviews WHERE id = $1", reviews[i].ID.Bytes)
			if err != nil {
				return nil, err
			}

			hasDeleted = true
		}
	}

	if hasDeleted {
		// We deleted some reviews, so we need to re-run the GC step to make sure we didn't orphan any reviews
		return GarbageCollect(ctx, okReviews)
	}

	return okReviews, nil
}
