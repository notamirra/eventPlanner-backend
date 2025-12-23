package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eventplanner-backend/internal/models"
	"eventplanner-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type EventHandler struct {
	events services.EventService
}

type createTaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	AssigneeID  *int       `json:"assigneeId,omitempty"`
}

func NewEventHandler(events services.EventService) *EventHandler {
	return &EventHandler{events: events}
}

func (h *EventHandler) Create(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req models.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startTime, use RFC3339"})
		return
	}
	e, err := h.events.Create(c, req.Title, req.Description, req.Location, start, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *EventHandler) ListOrganized(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	items, err := h.events.ListOrganized(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *EventHandler) ListInvited(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	items, err := h.events.ListInvited(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *EventHandler) Invite(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	var req models.InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Allow self-invitation
	if userID == req.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot invite yourself"})
		return
	}
	if err := h.events.Invite(c, eventID, userID, req.UserID, req.Role); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User invited successfully"})
}

func (h *EventHandler) Delete(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	if err := h.events.Delete(c, eventID, userID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
}

// AcceptInvite handles accepting an event invitation
func (h *EventHandler) AcceptInvite(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

	// Use the service layer to update attendance
	err = h.events.SetAttendance(c, eventID, userID, "going")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted successfully"})
}

func (h *EventHandler) Participants(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	items, err := h.events.Participants(c, eventID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateTask creates a new task for an event
// @Summary Create a task
// @Description Create a new task for an event (organizer only)
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Event ID"
// @Param request body createTaskRequest true "Task details"
// @Security ApiKeyAuth
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /events/{id}/tasks [post]
func (h *EventHandler) CreateTask(c *gin.Context) {
	userID := c.GetInt("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.events.CreateTask(
		c.Request.Context(),
		eventID,
		userID,
		req.Title,
		req.Description,
		req.DueDate,
		req.AssigneeID,
	)

	if err != nil {
		status := http.StatusInternalServerError
		errMsg := err.Error()
		
		if errMsg == "only organizers can create tasks" {
			status = http.StatusForbidden
		} else if strings.Contains(errMsg, "violates foreign key constraint") {
			status = http.StatusNotFound
			errMsg = "event or assignee not found"
		}

		c.JSON(status, gin.H{"error": errMsg})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *EventHandler) SetAttendance(c *gin.Context) {
	requesterID := c.GetInt("userID")
	if requesterID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

	var req struct {
		UserID int    `json:"userId"`
		Status string `json:"status" binding:"required,oneof=going maybe not_going"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetUserID := req.UserID
	if targetUserID == 0 {
		targetUserID = requesterID
	}

	// Users can only update their own attendance
	if targetUserID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only update your own attendance"})
		return
	}

	if err := h.events.SetAttendance(c, eventID, targetUserID, req.Status); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance updated successfully"})
}
