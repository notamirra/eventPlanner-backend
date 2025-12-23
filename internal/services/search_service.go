package services

import (
	"context"
	"time"

	"eventplanner-backend/internal/models"
	"eventplanner-backend/internal/repositories"
)

type SearchService interface {
	Search(ctx context.Context, userID int, q string, from, to *time.Time, role string) ([]models.Event, []models.Task, error)
}

type searchService struct {
	events repositories.EventRepository
}

func NewSearchService(events repositories.EventRepository) SearchService {
	return &searchService{events: events}
}

func (s *searchService) Search(ctx context.Context, userID int, q string, from, to *time.Time, role string) ([]models.Event, []models.Task, error) {
	return s.events.Search(ctx, userID, q, from, to, role)
}
