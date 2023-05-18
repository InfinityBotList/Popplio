package stafftemplates

import "popplio/state"

func Setup() {
	for _, templ := range StaffTemplates.Templates {
		appValidator := state.Validator.Struct(templ)

		if appValidator != nil {
			panic("App validation failed: " + appValidator.Error())
		}
	}
}
