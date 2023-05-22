package utils

import (
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"
)

type CreatePagedResult[T any] struct {
	Count   uint64
	Page    uint64
	PerPage uint64
	Path    string
	Query   []string // Optional
	Results []T
}

func CreatePage[T any](c CreatePagedResult[T]) types.PagedResult[T] {
	var previous string

	if c.Page > 2 {
		previous = state.Config.Sites.API + c.Path + "?page=" + strconv.FormatUint(c.Page-1, 10)

		if len(c.Query) > 0 {
			previous += "&" + strings.Join(c.Query, "&")
		}
	}

	var next string
	if c.Page+1 <= c.Count/c.PerPage {
		next = state.Config.Sites.API + c.Path + "?page=" + strconv.FormatUint(c.Page+1, 10)

		if len(c.Query) > 0 {
			next += "&" + strings.Join(c.Query, "&")
		}
	}

	return types.PagedResult[T]{
		Count:    c.Count,
		Results:  c.Results,
		PerPage:  c.PerPage,
		Previous: previous,
		Next:     next,
	}
}
