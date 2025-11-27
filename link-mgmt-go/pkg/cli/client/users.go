package client

import (
	"fmt"
	"net/http"

	"link-mgmt-go/pkg/models"
)

// CreateUserRequest represents the request payload for creating a user
type CreateUserRequest struct {
	Email string `json:"email"`
}

// CreateUser creates a new user and returns the user with API key
func (c *Client) CreateUser(email string) (*models.User, error) {
	var user models.User
	payload := CreateUserRequest{Email: email}
	if err := c.doJSONRequest(http.MethodPost, "/api/v1/users", payload, &user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}
