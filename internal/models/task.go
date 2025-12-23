package models

import "time"

type Task struct {
	ID         int       `json:"id"`
	EventID    int       `json:"eventId"`
	Title      string    `json:"title"`
	Description string   `json:"description"`
	DueDate    *time.Time `json:"dueDate"`
	AssigneeID *int      `json:"assigneeId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
