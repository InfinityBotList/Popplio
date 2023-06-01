package stafftemplates

import "popplio/state"

func Setup() {
	for _, templ := range StaffTemplates.Templates {
		templValidator := state.Validator.Struct(templ)

		if templValidator != nil {
			panic("Template validation failed: " + templValidator.Error())
		}
	}
}
