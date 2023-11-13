package types

import "github.com/jackc/pgx/v5/pgtype"

// @ci table=tasks
//
// Tasks are background processes that can be run on the server.
type Task struct {
	TaskId    string             `db:"task_id" json:"task_id" validate:"required" description:"The task ID."`
	TaskName  string             `db:"task_name" json:"task_name" validate:"required" description:"The task name."`
	Output    map[string]any     `db:"output" json:"output" description:"The task output."`
	Statuses  []map[string]any   `db:"statuses" json:"statuses" validate:"required" description:"The task statuses."`
	ForUser   pgtype.Text        `db:"for_user" json:"for_user" description:"The user this task is for."`
	Expiry    pgtype.Interval    `db:"expiry" json:"expiry" validate:"required" description:"The task expiry."`
	Status    string             `db:"status" json:"status" validate:"required" description:"The task status."`
	CreatedAt pgtype.Timestamptz `db:"created_at" json:"created_at" description:"The time the task was created."`
}
