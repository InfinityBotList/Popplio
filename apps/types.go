package apps

import "time"

type LogicFunc = func(p Position, answers map[string]string) error

type Question struct {
	ID          string `json:"id" validate:"required"`
	Question    string `json:"question" validate:"required"`
	Paragraph   string `json:"paragraph" validate:"required"`
	Placeholder string `json:"placeholder" validate:"required"`
	Short       bool   `json:"short" validate:"required"`
}

type Position struct {
	Order      int        `json:"order" validate:"required"`
	Info       string     `json:"info" validate:"required"`
	Name       string     `json:"name" validate:"required"`
	Interview  []Question `json:"interview"` // Optional as interview may not be required
	Questions  []Question `json:"questions" validate:"gt=0,required"`
	Hidden     bool       `json:"hidden"`
	ExtraLogic LogicFunc  `json:"-"`
	Closed     bool       `json:"closed"`
}

type AppMeta struct {
	Positions map[string]Position `json:"positions"`
	Stable    bool                `json:"stable"` // Stable means that the list of apps is not pending big changes
}

type Response struct {
	AppID     string            `json:"app_id"`
	UserID    string            `json:"user_id"`
	Answers   map[string]string `json:"answers"`
	Interview map[string]string `json:"interview"`
	State     string            `json:"state"`
	CreatedAt time.Time         `json:"created_at"`
	Likes     []string          `json:"likes"`
	Dislikes  []string          `json:"dislikes"`
	Position  string            `json:"position"`
}
