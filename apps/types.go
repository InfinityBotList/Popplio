package apps

import (
	"popplio/api"
	"time"
)

type LogicFunc = func(d api.RouteData, p Position, answers map[string]string) (add bool, err error)
type PositionDescriptionFunc = func(d api.RouteData, p Position) string

type Question struct {
	ID          string `json:"id" validate:"required"`
	Question    string `json:"question" validate:"required"`
	Paragraph   string `json:"paragraph" validate:"required"`
	Placeholder string `json:"placeholder" validate:"required"`
	Short       bool   `json:"short" validate:"required"`
}

type Position struct {
	Order     int        `json:"order" validate:"required"`
	Tags      []string   `json:"tags" validate:"required"`
	Info      string     `json:"info" validate:"required"`
	Name      string     `json:"name" validate:"required"`
	Interview []Question `json:"interview"` // Optional as interview may not be required
	Questions []Question `json:"questions" validate:"gt=0,required"`
	Hidden    bool       `json:"hidden"`
	Closed    bool       `json:"closed"`

	// Internal fields
	Channel             func() string           `json:"-"`
	ExtraLogic          LogicFunc               `json:"-"`
	PositionDescription PositionDescriptionFunc `json:"-"` // Used for custom position descriptions
	AllowedForBanned    bool                    `json:"-"` // If true, banned users can apply for this position
}

type AppMeta struct {
	Positions map[string]Position `json:"positions"`
	Stable    bool                `json:"stable"` // Stable means that the list of apps is not pending big changes
}

type AppResponse struct {
	AppID            string         `db:"app_id" json:"app_id"`
	UserID           string         `db:"user_id" json:"user_id"`
	Answers          map[string]any `db:"answers" json:"answers"`
	InterviewAnswers map[string]any `db:"interview_answers" json:"interview_answers"`
	State            string         `db:"state" json:"state"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	Position         string         `db:"position" json:"position"`
}

type AppList struct {
	AppID     string    `db:"app_id" json:"app_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	State     string    `db:"state" json:"state"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Position  string    `db:"position" json:"position"`
}

type AppListResponse struct {
	Apps []AppList `json:"apps"`
}
