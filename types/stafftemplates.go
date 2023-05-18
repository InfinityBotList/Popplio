package types

type StaffTemplateList struct {
	Templates []StaffTemplateMeta `json:"templates" validate:"required"`
}

type StaffTemplateMeta struct {
	Name        string          `json:"name" validate:"required"`
	Icon        string          `json:"icon" validate:"required"`
	Description string          `json:"description" validate:"required"`
	Templates   []StaffTemplate `json:"templates" validate:"required"`
}

type StaffTemplate struct {
	Name        string   `json:"name" validate:"required"`
	Emoji       string   `json:"emoji" validate:"required"`
	Tags        []string `json:"tags" validate:"required"`
	Description string   `json:"description" validate:"required"`
}
