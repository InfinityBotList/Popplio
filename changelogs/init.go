package changelogs

import "popplio/state"

func Setup() {
	for _, templ := range Changelog.Entries {
		clValidator := state.Validator.Struct(templ)

		if clValidator != nil {
			panic("Changelog validation failed: " + clValidator.Error())
		}
	}
}
