package db

import (
	"context"
	"fmt"

	"link-mgmt-go/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetUserByAPIKey retrieves a user by their API key
func (db *DB) GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	var user models.User
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, api_key, created_at, updated_at
		 FROM users WHERE api_key = $1`,
		apiKey,
	).Scan(
		&user.ID,
		&user.Email,
		&user.APIKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, apiKey string) (*models.User, error) {
	var user models.User
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, api_key)
		 VALUES ($1, $2)
		 RETURNING id, email, api_key, created_at, updated_at`,
		email, apiKey,
	).Scan(
		&user.ID,
		&user.Email,
		&user.APIKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetLinksByUserID retrieves all links for a user
func (db *DB) GetLinksByUserID(ctx context.Context, userID uuid.UUID) ([]models.Link, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, url, title, description, text, created_at, updated_at
		 FROM links
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query links: %w", err)
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var link models.Link
		err := rows.Scan(
			&link.ID,
			&link.UserID,
			&link.URL,
			&link.Title,
			&link.Description,
			&link.Text,
			&link.CreatedAt,
			&link.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// CreateLink creates a new link
func (db *DB) CreateLink(ctx context.Context, userID uuid.UUID, link models.LinkCreate) (*models.Link, error) {
	var created models.Link
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO links (user_id, url, title, description, text)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, url, title, description, text, created_at, updated_at`,
		userID, link.URL, link.Title, link.Description, link.Text,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.URL,
		&created.Title,
		&created.Description,
		&created.Text,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	return &created, nil
}

// GetLinkByID retrieves a link by ID
func (db *DB) GetLinkByID(ctx context.Context, linkID, userID uuid.UUID) (*models.Link, error) {
	var link models.Link
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, url, title, description, text, created_at, updated_at
		 FROM links
		 WHERE id = $1 AND user_id = $2`,
		linkID, userID,
	).Scan(
		&link.ID,
		&link.UserID,
		&link.URL,
		&link.Title,
		&link.Description,
		&link.Text,
		&link.CreatedAt,
		&link.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("link not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}

	return &link, nil
}

// UpdateLink updates an existing link
func (db *DB) UpdateLink(ctx context.Context, linkID, userID uuid.UUID, update models.LinkUpdate) (*models.Link, error) {
	// Build dynamic update query based on provided fields
	query := `UPDATE links SET updated_at = NOW()`
	args := []interface{}{linkID, userID}
	argPos := 3 // Start at $3 (after $1=linkID, $2=userID)

	if update.URL != nil {
		query += fmt.Sprintf(", url = $%d", argPos)
		args = append(args, *update.URL)
		argPos++
	}
	if update.Title != nil {
		query += fmt.Sprintf(", title = $%d", argPos)
		args = append(args, *update.Title)
		argPos++
	}
	if update.Description != nil {
		query += fmt.Sprintf(", description = $%d", argPos)
		args = append(args, *update.Description)
		argPos++
	}
	if update.Text != nil {
		query += fmt.Sprintf(", text = $%d", argPos)
		args = append(args, *update.Text)
		argPos++
	}

	query += ` WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, url, title, description, text, created_at, updated_at`

	var link models.Link
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&link.ID,
		&link.UserID,
		&link.URL,
		&link.Title,
		&link.Description,
		&link.Text,
		&link.CreatedAt,
		&link.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("link not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update link: %w", err)
	}

	return &link, nil
}

// DeleteLink deletes a link
func (db *DB) DeleteLink(ctx context.Context, linkID, userID uuid.UUID) error {
	result, err := db.Pool.Exec(ctx,
		`DELETE FROM links WHERE id = $1 AND user_id = $2`,
		linkID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("link not found")
	}

	return nil
}
