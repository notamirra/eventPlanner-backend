package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"eventplanner-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	search services.SearchService
}

// SearchResponse represents the enhanced search response structure
type SearchResponse struct {
	Meta      SearchMeta       `json:"meta"`
	Events    []EventResponse  `json:"events,omitempty"`
	Tasks     []TaskResponse   `json:"tasks,omitempty"`
}

type SearchMeta struct {
	Query       string     `json:"query,omitempty"`
	Role        string     `json:"role,omitempty"`
	DateRange   DateRange  `json:"dateRange,omitempty"`
	ResultCount int        `json:"resultCount"`
}

type DateRange struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

type EventResponse struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Location    string     `json:"location,omitempty"`
	StartTime   time.Time  `json:"startTime"`
	OrganizerID int        `json:"organizerId"`
	TimeUntil   string     `json:"timeUntil,omitempty"`
	IsUpcoming  bool       `json:"isUpcoming"`
}

type TaskResponse struct {
	ID          int        `json:"id"`
	EventID     int        `json:"eventId"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	Status      string     `json:"status"` // "upcoming", "today", "overdue"
}

func NewSearchHandler(search services.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// @Summary Search events and tasks (Authenticated)
// @Description Search for events and tasks with role-based filtering. Requires authentication.
// @Tags search
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param query query string false "Search query (searches in title, description, location)"
// @Param q query string false "Legacy parameter, use 'query' instead"
// @Param start query string false "Start date (format: YYYY-MM-DD or 'today')"
// @Param from query string false "Legacy parameter, use 'start' instead"
// @Param end query string false "End date (format: YYYY-MM-DD or 'today')"
// @Param to query string false "Legacy parameter, use 'end' instead"
// @Param userRole query string true "Filter by role (must be one of: organizer, attendee, collaborator)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /search [get]
func (h *SearchHandler) Search(c *gin.Context) {
	// Require authentication
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Parse query parameters (support both new and legacy parameter names)
	q := strings.TrimSpace(c.DefaultQuery("query", c.Query("q")))
	role := strings.ToLower(strings.TrimSpace(c.DefaultQuery("userRole", c.Query("role"))))

	// If role is empty, treat it as "allroles"
	if role == "" {
		role = "allroles"
	}

	// Validate role parameter
	validRoles := map[string]bool{
		"organizer":    true,
		"attendee":     true,
		"collaborator": true,
		"allroles":     true,
	}
	if !validRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "valid role parameter is required (allroles, organizer, attendee, or collaborator)",
		})
		return
	}
	
	// Parse date range with support for special values
	now := time.Now()
	truncateToDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	// Helper function to parse date with special values
	parseDate := func(dateStr string) (*time.Time, error) {
		switch strings.ToLower(dateStr) {
		case "today":
			t := truncateToDay(now)
			return &t, nil
		case "tomorrow":
			t := truncateToDay(now.Add(24 * time.Hour))
			return &t, nil
		case "nextweek":
			t := truncateToDay(now.Add(7 * 24 * time.Hour))
			return &t, nil
		default:
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return nil, err
			}
			return &t, nil
		}
	}

	// Parse start date (from query parameter or legacy 'from' parameter)
	var fromPtr, toPtr *time.Time
	if startParam := c.DefaultQuery("start", c.Query("from")); startParam != "" {
		t, err := parseDate(startParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'start' date format, use YYYY-MM-DD or 'today'"})
			return
		}
		fromPtr = t
	}

	// Parse end date (from query parameter or legacy 'to' parameter)
	if endParam := c.DefaultQuery("end", c.Query("to")); endParam != "" {
		t, err := parseDate(endParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'end' date format, use YYYY-MM-DD or 'today'"})
			return
		}
		// Set to end of day
		truncated := t.Truncate(24 * time.Hour).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		toPtr = &truncated
	}

	// Execute search with the authenticated user's ID
	events, tasks, err := h.search.Search(c, userID.(int), q, fromPtr, toPtr, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform search"})
		return
	}

	// Prepare response with enhanced format
	var eventResults []map[string]interface{}
	var taskResults []map[string]interface{}

	// Process events
	for _, e := range events {
		eventData := map[string]interface{}{
			"id":          e.ID,
			"title":       e.Title,
			"description": e.Description,
			"location":    e.Location,
			"startTime":   e.StartTime,
			"organizerId": e.OrganizerID,
			"isUpcoming":  e.StartTime.After(now),
		}

		// Add time until event if it's upcoming
		if e.StartTime.After(now) {
			duration := e.StartTime.Sub(now)
			hours := int(duration.Hours())
			days := hours / 24

			switch {
			case days > 30:
				eventData["timeUntil"] = "in more than a month"
			case days > 1:
				eventData["timeUntil"] = "in " + fmt.Sprintf("%d days", days)
			case hours >= 1:
				eventData["timeUntil"] = "in " + fmt.Sprintf("%d hours", hours)
			default:
				eventData["timeUntil"] = "very soon"
			}
		}

		eventResults = append(eventResults, eventData)
	}

	// Process tasks
	for _, t := range tasks {
		taskData := map[string]interface{}{
			"id":          t.ID,
			"eventId":     t.EventID,
			"title":       t.Title,
			"description": t.Description,
			"dueDate":     t.DueDate,
			"assigneeId":  t.AssigneeID,
		}

		// Set task status
		if t.DueDate != nil {
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			tomorrow := today.Add(24 * time.Hour)

			switch {
			case t.DueDate.Before(today):
				taskData["status"] = "overdue"
			case t.DueDate.Before(tomorrow):
				taskData["status"] = "today"
			default:
				taskData["status"] = "upcoming"
			}
		} else {
			taskData["status"] = "no-due-date"
		}

		taskResults = append(taskResults, taskData)
	}

	// Return response in the expected format
	response := map[string]interface{}{
		"events": eventResults,
		"tasks":  taskResults,
	}

	c.JSON(http.StatusOK, response)
}
