package seo

import "github.com/jackc/pgx/v5"

// RSS/Sitemaps often have the common pattern of collecting a single id field
//
// This is a helper function to collect those ids
type IDCollector struct {
	Added map[string]bool
}

func (i *IDCollector) Collect(rows pgx.Rows) ([]string, error) {
	if i.Added == nil {
		i.Added = map[string]bool{}
	}

	var toAdd []string
	for rows.Next() {
		var id string

		err := rows.Scan(&id)

		if err != nil {
			return nil, err
		}

		if i.Added[id] {
			continue
		}

		i.Added[id] = true

		toAdd = append(toAdd, id)
	}

	return toAdd, nil
}
