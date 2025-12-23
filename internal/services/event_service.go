package services

import (
	"context"
	"fmt"
	"time"

	"eventplanner-backend/internal/models"
	"eventplanner-backend/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type EventService interface {
	Create(ctx context.Context, title, description, location string, start time.Time, organizerID int) (*models.Event, error)
	ListOrganized(ctx context.Context, userID int) ([]models.Event, error)
	ListInvited(ctx context.Context, userID int) ([]models.Event, error)
	Delete(ctx context.Context, eventID, organizerID int) error
	Invite(ctx context.Context, eventID, inviterID, inviteeID int, role string) error
	Participants(ctx context.Context, eventID, requesterID int) ([]models.Participant, error)
	SetAttendance(ctx context.Context, eventID, userID int, status string) error
	IsOrganizer(ctx context.Context, eventID, userID int) (bool, error)
	CreateTask(ctx context.Context, eventID, userID int, title, description string, dueDate *time.Time, assigneeID *int) (*models.Task, error)
}

type eventService struct {
	repo repositories.EventRepository
}

func NewEventService(repo repositories.EventRepository) EventService {
	return &eventService{repo: repo}
}

func (s *eventService) Create(ctx context.Context, title, description, location string, start time.Time, organizerID int) (*models.Event, error) {
	return s.repo.Create(ctx, title, description, location, start, organizerID)
}

func (s *eventService) ListOrganized(ctx context.Context, userID int) ([]models.Event, error) {
	return s.repo.ListByRole(ctx, userID, "organizer")
}

func (s *eventService) ListInvited(ctx context.Context, userID int) ([]models.Event, error) {
	return s.repo.ListByRole(ctx, userID, "attendee")
}

func (s *eventService) Delete(ctx context.Context, eventID, organizerID int) error {
	return s.repo.DeleteIfOrganizer(ctx, eventID, organizerID)
}

func (s *eventService) Invite(ctx context.Context, eventID, inviterID, inviteeID int, role string) error {
	return s.repo.Invite(ctx, eventID, inviterID, inviteeID, role)
}

func (s *eventService) Participants(ctx context.Context, eventID, requesterID int) ([]models.Participant, error) {
    ok, err := s.repo.IsOrganizer(ctx, eventID, requesterID)
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, pgx.ErrNoRows
    }
    return s.repo.ListParticipants(ctx, eventID)
}

func (s *eventService) SetAttendance(ctx context.Context, eventID, userID int, status string) error {
	return s.repo.SetAttendance(ctx, eventID, userID, status)
}

func (s *eventService) IsOrganizer(ctx context.Context, eventID, userID int) (bool, error) {
	return s.repo.IsOrganizer(ctx, eventID, userID)
}

func (s *eventService) CreateTask(ctx context.Context, eventID, userID int, title, description string, dueDate *time.Time, assigneeID *int) (*models.Task, error) {
	// Check if the user is an organizer of the event
	isOrganizer, err := s.repo.IsOrganizer(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow organizers to create tasks
	if !isOrganizer {
		return nil, fmt.Errorf("only organizers can create tasks")
	}

	// Validate required fields
	if title == "" {
		return nil, fmt.Errorf("task title is required")
	}

	// Create the task
	task, err := s.repo.CreateTask(ctx, eventID, title, description, dueDate, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}
