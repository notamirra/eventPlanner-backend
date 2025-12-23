package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"eventplanner-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(ctx context.Context, title, description, location string, start time.Time, organizerID int) (*models.Event, error)
	ListByRole(ctx context.Context, userID int, role string) ([]models.Event, error)
	DeleteIfOrganizer(ctx context.Context, eventID, organizerID int) error
	Invite(ctx context.Context, eventID, inviterID, inviteeID int, role string) error
	ListParticipants(ctx context.Context, eventID int) ([]models.Participant, error)
	SetAttendance(ctx context.Context, eventID, userID int, status string) error
	Search(ctx context.Context, userID int, q string, from, to *time.Time, role string) ([]models.Event, []models.Task, error)
	IsOrganizer(ctx context.Context, eventID, userID int) (bool, error)
	CreateTask(ctx context.Context, eventID int, title, description string, dueDate *time.Time, assigneeID *int) (*models.Task, error)
}

func (r *eventRepository) IsOrganizer(ctx context.Context, eventID, userID int) (bool, error) {
    const q = `SELECT 1 FROM event_participants WHERE event_id=$1 AND user_id=$2 AND role='organizer'`
    if err := r.pool.QueryRow(ctx, q, eventID, userID).Scan(new(int)); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

type eventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

func (r *eventRepository) checkExistingEvent(ctx context.Context, start time.Time) (bool, error) {
    const q = `SELECT EXISTS(SELECT 1 FROM events WHERE start_time = $1)`
    var exists bool
    err := r.pool.QueryRow(ctx, q, start).Scan(&exists)
    return exists, err
}

func (r *eventRepository) Create(ctx context.Context, title, description, location string, start time.Time, organizerID int) (*models.Event, error) {
    // Check for existing event at the same time
    exists, err := r.checkExistingEvent(ctx, start)
    if err != nil {
        return nil, fmt.Errorf("error checking for existing event: %w", err)
    }
    if exists {
        return nil, fmt.Errorf("an event already exists at this time")
    }

    const q = `
        INSERT INTO events (title, description, location, start_time, organizer_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, title, description, location, start_time, organizer_id, created_at, updated_at
    `

    var event models.Event
    err = r.pool.QueryRow(
        ctx,
        q,
        title,
        description,
        location,
        start,
        organizerID,
    ).Scan(
        &event.ID,
        &event.Title,
        &event.Description,
        &event.Location,
        &event.StartTime,
        &event.OrganizerID,
        &event.CreatedAt,
        &event.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    // Add organizer as participant
    if _, err := r.pool.Exec(
        ctx,
        `INSERT INTO event_participants (event_id, user_id, role) VALUES ($1, $2, 'organizer')`,
        event.ID,
        organizerID,
    ); err != nil {
        return nil, err
    }

    return &event, nil
}

func (r *eventRepository) ListByRole(ctx context.Context, userID int, role string) ([]models.Event, error) {
	const q = `
		SELECT e.id, e.title, e.description, e.location, e.start_time, e.organizer_id, e.created_at, e.updated_at
		FROM events e
		JOIN event_participants p ON p.event_id = e.id
		WHERE p.user_id = $1 AND p.role = $2
		ORDER BY e.start_time ASC
	`
	rows, err := r.pool.Query(ctx, q, userID, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &e.StartTime, &e.OrganizerID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}

func (r *eventRepository) DeleteIfOrganizer(ctx context.Context, eventID, organizerID int) error {
	const check = `SELECT 1 FROM event_participants WHERE event_id=$1 AND user_id=$2 AND role='organizer'`
	if err := r.pool.QueryRow(ctx, check, eventID, organizerID).Scan(new(int)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id=$1`, eventID)
	return err
}

func (r *eventRepository) Invite(ctx context.Context, eventID, inviterID, inviteeID int, role string) error {
	const check = `SELECT 1 FROM event_participants WHERE event_id=$1 AND user_id=$2 AND role='organizer'`
	if err := r.pool.QueryRow(ctx, check, eventID, inviterID).Scan(new(int)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	const insert = `
		INSERT INTO event_participants (event_id, user_id, role, invited_by)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (event_id,user_id) DO UPDATE SET role=EXCLUDED.role, invited_by=EXCLUDED.invited_by, updated_at=now()
	`
	_, err := r.pool.Exec(ctx, insert, eventID, inviteeID, strings.ToLower(role), inviterID)
	return err
}

func (r *eventRepository) ListParticipants(ctx context.Context, eventID int) ([]models.Participant, error) {
	const q = `
		SELECT p.event_id, p.user_id, u.name, u.email, p.role, p.attendance
		FROM event_participants p
		JOIN users u ON u.id = p.user_id
		WHERE p.event_id = $1
		ORDER BY u.name
	`
	rows, err := r.pool.Query(ctx, q, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Participant
	for rows.Next() {
		var p models.Participant
		var attendance *string
		if err := rows.Scan(&p.EventID, &p.UserID, &p.UserName, &p.UserEmail, &p.Role, &attendance); err != nil {
			return nil, err
		}
		p.Attendance = attendance
		res = append(res, p)
	}
	return res, rows.Err()
}

func (r *eventRepository) SetAttendance(ctx context.Context, eventID, userID int, status string) error {
	// First check if the user is already a participant
	var exists bool
	err := r.pool.QueryRow(ctx, 
		`SELECT EXISTS(SELECT 1 FROM event_participants WHERE event_id=$1 AND user_id=$2)`, 
		eventID, userID).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// If not a participant, insert them as an attendee with the given status
		_, err = r.pool.Exec(ctx, `
			INSERT INTO event_participants (event_id, user_id, role, attendance, updated_at)
			VALUES ($1, $2, 'attendee', $3, NOW())
		`, eventID, userID, strings.ToLower(status))
		return err
	}

	// Update existing attendance
	_, err = r.pool.Exec(ctx, `
		UPDATE event_participants 
		SET attendance = $3, 
			updated_at = NOW()
		WHERE event_id = $1 AND user_id = $2
	`, eventID, userID, strings.ToLower(status))

	return err
}

func (r *eventRepository) Search(ctx context.Context, userID int, q string, from, to *time.Time, role string) ([]models.Event, []models.Task, error) {
	// Debug logging
	log.Printf("Search params - userID: %d, query: '%s', from: %v, to: %v, role: '%s'", userID, q, from, to, role)
	
	var econds []string
	var eargs []any
	idx := 1

	// If user is not 0 (meaning we have an authenticated user) and role is specified and not 'allroles'
	if userID != 0 && role != "" && role != "allroles" {
		// For authenticated users with role filter
		if role == "organizer" {
			econds = append(econds, "e.organizer_id = $"+itoa(idx))
			eargs = append(eargs, userID)
			idx++
		} else {
			econds = append(econds, "p.user_id = $"+itoa(idx)+" AND p.role = $"+itoa(idx+1))
			eargs = append(eargs, userID, strings.ToLower(role))
			idx += 2
		}
	}
	if q != "" {
		// When there's a search query, require both the query and time conditions to match
		var queryConds []string
		queryConds = append(queryConds, "(e.title ILIKE '%'||$"+itoa(idx)+"||'%' OR e.description ILIKE '%'||$"+itoa(idx)+"||'%' OR e.location ILIKE '%'||$"+itoa(idx)+"||'%')")
		tmpIdx := idx + 1
		
		// Add time conditions to the query conditions
		if from != nil {
			queryConds = append(queryConds, "e.start_time >= $"+itoa(tmpIdx))
			tmpIdx++
		}
		if to != nil {
			queryConds = append(queryConds, "e.start_time <= $"+itoa(tmpIdx))
			tmpIdx++
		}
		
		// Combine all conditions with AND and add them as a single condition
		econds = append(econds, "(" + strings.Join(queryConds, " AND ") + ")")
		
		// Add the query parameter
		eargs = append(eargs, q)
		
		// Add time parameters if they exist
		if from != nil {
			eargs = append(eargs, *from)
		}
		if to != nil {
			eargs = append(eargs, *to)
		}
		
		idx = tmpIdx
	} else {
		// Original behavior when there's no search query
		if from != nil {
			econds = append(econds, "e.start_time >= $"+itoa(idx))
			eargs = append(eargs, *from)
			idx++
		}
		if to != nil {
			econds = append(econds, "e.start_time <= $"+itoa(idx))
			eargs = append(eargs, *to)
			idx++
		}
	}
	// Build the base query
	baseQuery := `
		SELECT e.id, e.title, e.description, e.location, e.start_time, e.organizer_id, e.created_at, e.updated_at 
		FROM events e`

	// Add JOIN for participant checks if needed (for role-based filtering)
	if userID != 0 && role != "" && role != "allroles" && role != "organizer" {
		baseQuery += ` JOIN event_participants p ON p.event_id = e.id`
	}

	// Add WHERE clause if we have any conditions
	whereClause := ""
	if len(econds) > 0 {
		whereClause = " WHERE " + strings.Join(econds, " AND ")
	}

	// Final query with ordering
	qe := baseQuery + whereClause + ` ORDER BY e.start_time ASC`
	
	log.Printf("Events query: %s", qe)
	log.Printf("Query args: %v", eargs)
	
	rows, err := r.pool.Query(ctx, qe, eargs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &e.StartTime, &e.OrganizerID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	// Tasks
	idx = 1
	var tconds []string
	var targs []any
	
	// Start with the base task query that joins with events
	taskBaseQuery := `
		SELECT DISTINCT t.id, t.event_id, t.title, t.description, t.due_date, t.assignee_id, t.created_at, t.updated_at 
		FROM tasks t 
		JOIN events e ON e.id = t.event_id`

	// Add conditions based on role and search query
	if userID != 0 && role != "" {
		// For organizer role, show tasks from events they organize
		if role == "organizer" {
			tconds = append(tconds, "e.organizer_id = $"+itoa(idx))
			targs = append(targs, userID)
			idx++
		} else if role != "allroles" {
			// For other roles, show tasks assigned to users with that role
			taskBaseQuery += ` 
				JOIN event_participants ep ON ep.event_id = e.id AND ep.user_id = t.assignee_id`
			tconds = append(tconds, "t.assignee_id IS NOT NULL")
			tconds = append(tconds, "ep.role = $"+itoa(idx))
			targs = append(targs, strings.ToLower(role))
			idx++
		}
	}

	// Add search term condition if provided
	if q != "" {
		tconds = append(tconds, "(t.title ILIKE '%'||$"+itoa(idx)+"||'%' OR t.description ILIKE '%'||$"+itoa(idx)+"||'%' OR e.title ILIKE '%'||$"+itoa(idx)+"||'%' OR e.description ILIKE '%'||$"+itoa(idx)+"||'%')")
		targs = append(targs, q)
		idx++
	}
	if from != nil {
		tconds = append(tconds, "(t.due_date IS NULL OR t.due_date >= $"+itoa(idx)+")")
		targs = append(targs, *from)
		idx++
	}
	if to != nil {
		tconds = append(tconds, "(t.due_date IS NULL OR t.due_date <= $"+itoa(idx)+")")
		targs = append(targs, *to)
		idx++
	}
	// The taskBaseQuery is now defined at the top with the initial joins

	// Build the final query
	query := taskBaseQuery
	if len(tconds) > 0 {
		query += ` WHERE ` + strings.Join(tconds, ` AND `)
	}
	query += ` ORDER BY t.due_date NULLS LAST`
	
	// Debug logging
	log.Printf("Executing tasks query: %s", query)
	log.Printf("Query parameters: %v", targs)
	
	// Execute the query
	rows2, err := r.pool.Query(ctx, query, targs...)
	if err != nil {
		log.Printf("Error executing tasks query: %v", err)
		return events, nil, fmt.Errorf("error executing tasks query: %w", err)
	}
	defer rows2.Close()
	var tasks []models.Task
	for rows2.Next() {
		var t models.Task
		var due *time.Time
		var assignee *int
		if err := rows2.Scan(&t.ID, &t.EventID, &t.Title, &t.Description, &due, &assignee, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return events, nil, err
		}
		t.DueDate = due
		t.AssigneeID = assignee
		tasks = append(tasks, t)
	}
	return events, tasks, rows2.Err()
}

func (r *eventRepository) CreateTask(ctx context.Context, eventID int, title, description string, dueDate *time.Time, assigneeID *int) (*models.Task, error) {
	const q = `
		INSERT INTO tasks (event_id, title, description, due_date, assignee_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, event_id, title, description, due_date, assignee_id, created_at, updated_at
	`

	var task models.Task
	err := r.pool.QueryRow(
		ctx,
		q,
		eventID,
		title,
		description,
		dueDate,
		assigneeID,
	).Scan(
		&task.ID,
		&task.EventID,
		&task.Title,
		&task.Description,
		&task.DueDate,
		&task.AssigneeID,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &task, nil
}

func itoa(i int) string { return fmtInt(i) }

func fmtInt(i int) string {
	return fmt.Sprintf("%d", i)
}
