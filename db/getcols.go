// Package db derives SQL column lists from the struct tags on Popplio's
// types.
//
// Every type in popplio/types tags its fields with the database column they
// map to. GetCols turns those tags back into the column list a SELECT needs,
// so a query and the struct it scans into cannot drift apart when a field is
// added or renamed.
package db

import "reflect"

func GetCols(s any) []string {
	refType := reflect.TypeOf(s)

	var cols []string

	for _, f := range reflect.VisibleFields(refType) {
		db := f.Tag.Get("db")
		reflectOpts := f.Tag.Get("reflect")

		if db == "-" || db == "" || reflectOpts == "ignore" {
			continue
		}

		cols = append(cols, db)
	}

	return cols
}
