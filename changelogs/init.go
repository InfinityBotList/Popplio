package changelogs

import "popplio/state"

func Setup() {
	for _, entry := range Changelog.Entries {
		clValidator := state.Validator.Struct(entry)

		if clValidator != nil {
			panic("Changelog validation failed: " + clValidator.Error())
		}
	}
}
