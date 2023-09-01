package changelogs

import "popplio/state"

func Setup() {
	for _, entry := range Changelog.Entries {
		clValidator := state.Validator.Struct(entry)

		if clValidator != nil {
			panic("Changelog validation failed: " + clValidator.Error())
		}

		_, err := state.Pool.Exec(state.Context, "INSERT INTO changelogs (version, added, updated, removed) VALUES ($1, $2, $3, $4) ON CONFLICT (version) DO UPDATE SET added = $2, updated = $3, removed = $4", entry.Version, entry.Added, entry.Updated, entry.Removed)

		if err != nil {
			panic(err)
		}
	}
}
