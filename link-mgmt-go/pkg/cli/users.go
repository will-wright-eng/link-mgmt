package cli

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/config"
)

// RegisterUser creates a new user account and saves the API key
func (a *App) RegisterUser(email string) error {
	apiClient, err := a.getClientForRegistration()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	user, err := apiClient.CreateUser(email)
	if err != nil {
		// Check for common errors and provide helpful messages
		errStr := err.Error()
		if strings.Contains(errStr, "relation \"users\" does not exist") || strings.Contains(errStr, "does not exist") {
			return fmt.Errorf(`database table 'users' does not exist. Please run migrations first.

To run migrations:
  From project root (Docker):  make migrate`)
		}
		// Don't wrap the error again since it already contains "failed to register user"
		return err
	}

	// Save API key to config
	a.cfg.CLI.APIKey = user.APIKey
	if err := config.Save(a.cfg); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	// Update the client with the new API key
	a.client = client.NewClient(a.cfg.CLI.BaseURL, user.APIKey)

	fmt.Println("✓ User registered successfully!")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  User ID: %s\n", user.ID.String())
	fmt.Printf("  API key saved to config automatically\n")
	fmt.Println("\n⚠️  Save this API key securely (it won't be shown again):")
	fmt.Printf("  %s\n", user.APIKey)

	return nil
}
