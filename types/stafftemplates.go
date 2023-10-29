package types

import "time"

type StaffTemplateType struct {
	ID    string `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	Icon  string `db:"icon" json:"icon"`
	Short string `db:"short" json:"short"`
}

type StaffTemplate struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Emoji       string    `db:"emoji" json:"emoji"`
	Tags        []string  `db:"tags" json:"tags"`
	Description string    `db:"description" json:"description"`
	Type        string    `db:"type" json:"type"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type StaffTemplateList struct {
	TemplateTypes []StaffTemplateType `json:"template_types"`
	Templates     []StaffTemplate     `json:"templates"`
}
