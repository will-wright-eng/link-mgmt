package client

import (
	"fmt"
	"net/http"

	"link-mgmt-go/pkg/models"

	"github.com/google/uuid"
)

// ListLinks retrieves all links for the authenticated user
func (c *Client) ListLinks() ([]models.Link, error) {
	var links []models.Link
	if err := c.doGetRequest("/api/v1/links", &links); err != nil {
		return nil, err
	}
	return links, nil
}

// GetLink retrieves a specific link by ID
func (c *Client) GetLink(id uuid.UUID) (*models.Link, error) {
	var link models.Link
	path := fmt.Sprintf("/api/v1/links/%s", id.String())
	if err := c.doGetRequest(path, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

// CreateLink creates a new link
func (c *Client) CreateLink(link models.LinkCreate) (*models.Link, error) {
	var created models.Link
	if err := c.doJSONRequest(http.MethodPost, "/api/v1/links", link, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateLink updates an existing link
func (c *Client) UpdateLink(id uuid.UUID, update models.LinkUpdate) (*models.Link, error) {
	var updated models.Link
	path := fmt.Sprintf("/api/v1/links/%s", id.String())
	if err := c.doJSONRequest(http.MethodPut, path, update, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteLink deletes a link by ID
func (c *Client) DeleteLink(id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/links/%s", id.String())
	return c.doDeleteRequest(path)
}
