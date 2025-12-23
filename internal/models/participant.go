package models

type Participant struct {
	EventID    int     `json:"eventId"`
	UserID     int     `json:"userId"`
	UserName   string  `json:"userName"`
	UserEmail  string  `json:"userEmail"`
	Role       string  `json:"role"`
	Attendance *string `json:"attendance"`
}

type InviteRequest struct {
	UserID int    `json:"userId" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=organizer attendee collaborator"`
}

type AttendanceRequest struct {
	Status string `json:"status" binding:"required,oneof=going maybe not_going"`
}
