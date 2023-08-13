package types

import "time"

type ResolvedReminder struct {
	Name   string `db:"-" json:"name"`
	Avatar string `db:"-" json:"avatar"`
}

type Reminder struct {
	UserID     string            `db:"user_id" json:"user_id"`
	TargetType string            `db:"target_type" json:"target_type"`
	TargetID   string            `db:"target_id" json:"target_id"`
	Resolved   *ResolvedReminder `db:"-" json:"resolved"`
	CreatedAt  time.Time         `db:"created_at" json:"created_at"`
	LastAcked  time.Time         `db:"last_acked" json:"last_acked"`
}

type ReminderList struct {
	Reminders []Reminder `json:"reminders"`
}
