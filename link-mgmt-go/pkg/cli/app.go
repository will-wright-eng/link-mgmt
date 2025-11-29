package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"link-mgmt-go/pkg/cli/client"
	"link-mgmt-go/pkg/config"
	"link-mgmt-go/pkg/models"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
)

type App struct {
	cfg    *config.Config
	client *client.Client
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

// getClient returns the HTTP client, creating it if necessary
func (a *App) getClient() (*client.Client, error) {
	if a.client != nil {
		return a.client, nil
	}

	if a.cfg.CLI.BaseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set cli.base_url)")
	}
	if a.cfg.CLI.APIKey == "" {
		return nil, fmt.Errorf("API key not configured")
	}

	a.client = client.NewClient(a.cfg.CLI.BaseURL, a.cfg.CLI.APIKey)
	return a.client, nil
}

// getClientForRegistration returns an HTTP client without API key (for registration)
func (a *App) getClientForRegistration() (*client.Client, error) {
	if a.cfg.CLI.BaseURL == "" {
		return nil, fmt.Errorf("base URL not configured (set cli.base_url)")
	}
	// Use empty API key for registration endpoint (doesn't require auth)
	return client.NewClient(a.cfg.CLI.BaseURL, ""), nil
}

// ShowConfig displays the current configuration
func (a *App) ShowConfig() {
	data, err := toml.Marshal(a.cfg)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// SetConfig sets a configuration value
// Format: section.key=value (e.g., "database.url=postgres://...")
func (a *App) SetConfig(setStr string) error {
	parts := strings.SplitN(setStr, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'section.key=value'")
	}

	keyPath := strings.Split(parts[0], ".")
	value := parts[1]

	if len(keyPath) != 2 {
		return fmt.Errorf("invalid key format: expected 'section.key'")
	}

	section := keyPath[0]
	key := keyPath[1]

	switch section {
	case "database":
		switch key {
		case "url":
			a.cfg.Database.URL = value
		default:
			return fmt.Errorf("unknown database key: %s", key)
		}
	case "api":
		switch key {
		case "host":
			a.cfg.API.Host = value
		case "port":
			var port int
			if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
				return fmt.Errorf("invalid port value: %s", value)
			}
			a.cfg.API.Port = port
		default:
			return fmt.Errorf("unknown api key: %s", key)
		}
	case "cli":
		switch key {
		case "base_url":
			a.cfg.CLI.BaseURL = value
		case "api_key":
			a.cfg.CLI.APIKey = value
		case "scrape_timeout":
			var timeout int
			if _, err := fmt.Sscanf(value, "%d", &timeout); err != nil {
				return fmt.Errorf("invalid scrape_timeout value: %s", value)
			}
			a.cfg.CLI.ScrapeTimeout = timeout
		default:
			return fmt.Errorf("unknown cli key: %s", key)
		}
	default:
		return fmt.Errorf("unknown section: %s", section)
	}

	return config.Save(a.cfg)
}

func (a *App) Run() error {
	fmt.Println("Interactive TUI mode - coming soon!")
	fmt.Println("Use --list, --add, or --delete for now")
	return nil
}

func (a *App) ListLinks() {
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	links, err := apiClient.ListLinks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching links: %v\n", err)
		os.Exit(1)
	}

	if len(links) == 0 {
		fmt.Println("No links found.")
		return
	}

	// Display links in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tTitle\tCreated")
	fmt.Fprintln(w, "───\t───\t───\t───")

	for _, link := range links {
		title := ""
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		} else {
			title = "(no title)"
		}

		// Truncate URL if too long
		url := link.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}

		// Format date
		created := link.CreatedAt.Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			link.ID.String()[:8]+"...",
			url,
			title,
			created,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d link(s)\n", len(links))
}

// AddLink prompts the user for link information and creates a new link
func (a *App) AddLink() {
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run the add link form
	form := newAddLinkForm(apiClient)
	p := tea.NewProgram(form)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running form: %v\n", err)
		os.Exit(1)
	}
}

// DeleteLink prompts the user to select and delete a link
func (a *App) DeleteLink() {
	apiClient, err := a.getClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run the delete link selector
	selector := newDeleteLinkSelector(apiClient)
	p := tea.NewProgram(selector)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running selector: %v\n", err)
		os.Exit(1)
	}
}

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

// addLinkForm is a Bubble Tea model for the add link form
type addLinkForm struct {
	client     *client.Client
	urlInput   textinput.Model
	titleInput textinput.Model
	descInput  textinput.Model
	textInput  textarea.Model
	step       int // 0=URL, 1=Title, 2=Description, 3=Text, 4=Done
	err        error
	created    *models.Link
}

func newAddLinkForm(client *client.Client) *addLinkForm {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com"
	urlInput.Focus()
	urlInput.CharLimit = 2048
	urlInput.Width = 60

	titleInput := textinput.New()
	titleInput.Placeholder = "Optional title"
	titleInput.CharLimit = 255
	titleInput.Width = 60

	descInput := textinput.New()
	descInput.Placeholder = "Optional description"
	descInput.CharLimit = 1000
	descInput.Width = 60

	textInput := textarea.New()
	textInput.Placeholder = "Optional text content (multi-line)"
	textInput.SetWidth(60)
	textInput.SetHeight(5)
	textInput.CharLimit = 10000

	return &addLinkForm{
		client:     client,
		urlInput:   urlInput,
		titleInput: titleInput,
		descInput:  descInput,
		textInput:  textInput,
		step:       0,
	}
}

func (m *addLinkForm) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addLinkForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.step == 0 {
				// Validate URL
				urlStr := strings.TrimSpace(m.urlInput.Value())
				if urlStr == "" {
					m.err = fmt.Errorf("URL is required")
					return m, nil
				}
				if _, err := url.Parse(urlStr); err != nil {
					m.err = fmt.Errorf("invalid URL: %v", err)
					return m, nil
				}
				m.err = nil
				m.step = 1
				m.titleInput.Focus()
				return m, textinput.Blink
			} else if m.step == 1 {
				// Title is optional, move to description
				m.step = 2
				m.descInput.Focus()
				return m, textinput.Blink
			} else if m.step == 2 {
				// Description is optional, move to text
				m.step = 3
				m.textInput.Focus()
				return m, textarea.Blink
			} else if m.step == 3 {
				// Text is optional, submit the form
				return m, m.submit()
			}
		}

	case submitErrorMsg:
		m.err = msg.err
		return m, nil
	case submitSuccessMsg:
		m.created = msg.link
		m.step = 4
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case 0:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 2:
		m.descInput, cmd = m.descInput.Update(msg)
	case 3:
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m *addLinkForm) View() string {
	if m.step == 4 {
		// Success view
		if m.created != nil {
			title := ""
			if m.created.Title != nil && *m.created.Title != "" {
				title = *m.created.Title
			} else {
				title = "(no title)"
			}
			return fmt.Sprintf(
				"\n✓ Link created successfully!\n\n"+
					"  ID:          %s\n"+
					"  URL:         %s\n"+
					"  Title:       %s\n"+
					"  Created:     %s\n\n"+
					"Press any key to exit...",
				m.created.ID.String()[:8]+"...",
				m.created.URL,
				title,
				m.created.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		return "\n✓ Link created successfully!\n\nPress any key to exit..."
	}

	var s strings.Builder
	s.WriteString("\nAdd New Link\n\n")

	switch m.step {
	case 0:
		s.WriteString("URL (required):\n")
		s.WriteString(m.urlInput.View())
		if m.err != nil {
			s.WriteString(fmt.Sprintf("\n\n❌ %s", m.err))
		}
		s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
	case 1:
		s.WriteString("✓ URL: " + m.urlInput.Value() + "\n\n")
		s.WriteString("Title (optional, press Enter to skip):\n")
		s.WriteString(m.titleInput.View())
		s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
	case 2:
		s.WriteString("✓ URL: " + m.urlInput.Value() + "\n")
		titleVal := m.titleInput.Value()
		if titleVal != "" {
			s.WriteString("✓ Title: " + titleVal + "\n")
		}
		s.WriteString("\nDescription (optional, press Enter to skip):\n")
		s.WriteString(m.descInput.View())
		s.WriteString("\n\n(Press Enter to continue, Esc to cancel)")
	case 3:
		s.WriteString("✓ URL: " + m.urlInput.Value() + "\n")
		titleVal := m.titleInput.Value()
		if titleVal != "" {
			s.WriteString("✓ Title: " + titleVal + "\n")
		}
		descVal := m.descInput.Value()
		if descVal != "" {
			s.WriteString("✓ Description: " + descVal + "\n")
		}
		s.WriteString("\nText (optional, press Enter to submit, Esc to cancel):\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n(Press Enter to submit, Esc to cancel)")
	}

	return s.String()
}

func (m *addLinkForm) submit() tea.Cmd {
	return func() tea.Msg {
		urlStr := strings.TrimSpace(m.urlInput.Value())
		titleStr := strings.TrimSpace(m.titleInput.Value())
		descStr := strings.TrimSpace(m.descInput.Value())
		textStr := strings.TrimSpace(m.textInput.Value())

		linkCreate := models.LinkCreate{
			URL: urlStr,
		}

		if titleStr != "" {
			linkCreate.Title = &titleStr
		}
		if descStr != "" {
			linkCreate.Description = &descStr
		}
		if textStr != "" {
			linkCreate.Text = &textStr
		}

		created, err := m.client.CreateLink(linkCreate)
		if err != nil {
			return submitErrorMsg{err: err}
		}

		return submitSuccessMsg{link: created}
	}
}

type submitErrorMsg struct {
	err error
}

type submitSuccessMsg struct {
	link *models.Link
}

// deleteLinkSelector is a Bubble Tea model for selecting a link to delete
type deleteLinkSelector struct {
	client   *client.Client
	links    []models.Link
	selected int
	step     int // 0=selecting, 1=confirming, 2=done
	err      error
	confirm  textinput.Model
}

func newDeleteLinkSelector(client *client.Client) *deleteLinkSelector {
	confirm := textinput.New()
	confirm.Placeholder = "y/N"
	confirm.CharLimit = 1
	confirm.Width = 10

	return &deleteLinkSelector{
		client:  client,
		links:   []models.Link{},
		step:    0,
		confirm: confirm,
	}
}

func (m *deleteLinkSelector) Init() tea.Cmd {
	return m.loadLinks
}

func (m *deleteLinkSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case linksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.links = msg.links
		if len(m.links) == 0 {
			m.err = fmt.Errorf("no links available to delete")
			return m, tea.Quit
		}
		return m, nil

	case deleteErrorMsg:
		m.err = msg.err
		return m, tea.Quit
	case deleteSuccessMsg:
		m.step = 2
		return m, nil

	case tea.KeyMsg:
		if m.step == 0 {
			// Selection step
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
				return m, nil
			case "down", "j":
				if m.selected < len(m.links)-1 {
					m.selected++
				}
				return m, nil
			case "enter":
				if m.selected < len(m.links) {
					m.step = 1
					m.confirm.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}
		} else if m.step == 1 {
			// Confirmation step
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				answer := strings.ToLower(strings.TrimSpace(m.confirm.Value()))
				if answer == "y" || answer == "yes" {
					return m, m.deleteLink()
				}
				// Cancelled
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				m.confirm, cmd = m.confirm.Update(msg)
				return m, cmd
			}
		} else if m.step == 2 {
			// Done step - any key exits
			return m, tea.Quit
		}
	}

	if m.step == 1 {
		var cmd tea.Cmd
		m.confirm, cmd = m.confirm.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *deleteLinkSelector) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n\nPress any key to exit...", m.err)
	}

	if m.step == 2 {
		return "\n✓ Link deleted successfully!\n\nPress any key to exit..."
	}

	var s strings.Builder
	s.WriteString("\nSelect a link to delete:\n\n")

	if m.step == 0 {
		// Selection view
		for i, link := range m.links {
			marker := " "
			if i == m.selected {
				marker = "→"
			}

			title := ""
			if link.Title != nil && *link.Title != "" {
				title = *link.Title
			} else {
				title = "(no title)"
			}

			url := link.URL
			if len(url) > 50 {
				url = url[:47] + "..."
			}

			style := lipgloss.NewStyle()
			if i == m.selected {
				style = style.Bold(true)
			}

			s.WriteString(fmt.Sprintf("%s %s - %s\n", marker, style.Render(title), url))
		}
		s.WriteString("\n(Use ↑/↓ or j/k to navigate, Enter to select, Esc to cancel)")
	} else if m.step == 1 {
		// Confirmation view
		link := m.links[m.selected]
		title := ""
		if link.Title != nil && *link.Title != "" {
			title = *link.Title
		} else {
			title = "(no title)"
		}

		s.WriteString(fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", title))
		s.WriteString(fmt.Sprintf("URL: %s\n\n", link.URL))
		s.WriteString("Confirm (y/N): ")
		s.WriteString(m.confirm.View())
		s.WriteString("\n\n(Press Enter to confirm, Esc to cancel)")
	}

	return s.String()
}

func (m *deleteLinkSelector) loadLinks() tea.Msg {
	links, err := m.client.ListLinks()
	if err != nil {
		return linksLoadedMsg{err: err}
	}
	return linksLoadedMsg{links: links}
}

func (m *deleteLinkSelector) deleteLink() tea.Cmd {
	return func() tea.Msg {
		if m.selected >= len(m.links) {
			return deleteErrorMsg{err: fmt.Errorf("invalid selection")}
		}

		link := m.links[m.selected]
		err := m.client.DeleteLink(link.ID)
		if err != nil {
			return deleteErrorMsg{err: err}
		}

		return deleteSuccessMsg{}
	}
}

type linksLoadedMsg struct {
	links []models.Link
	err   error
}

type deleteErrorMsg struct {
	err error
}

type deleteSuccessMsg struct{}
