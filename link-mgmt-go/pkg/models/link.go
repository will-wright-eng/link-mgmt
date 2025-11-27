package models

import (
	"time"

	"github.com/google/uuid"
)

type Link struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UserID      uuid.UUID `db:"user_id" json:"user_id"`
	URL         string    `db:"url" json:"url"`
	Title       *string   `db:"title" json:"title,omitempty"`
	Description *string   `db:"description" json:"description,omitempty"`
	Text        *string   `db:"text" json:"text,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// LinkCreate represents data for creating a new link
type LinkCreate struct {
	URL         string  `json:"url" binding:"required"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Text        *string `json:"text,omitempty"`
}

// LinkUpdate represents data for updating a link
type LinkUpdate struct {
	URL         *string `json:"url,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Text        *string `json:"text,omitempty"`
}
