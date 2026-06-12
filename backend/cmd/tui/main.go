package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#6124DF")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

// Screen types
type screen int

const (
	screenDashboard screen = iota
	screenBackups
	screenDatabases
	screenSettings
)

// Key bindings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Refresh  key.Binding
	NewBackup key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	NewBackup: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new backup"),
	),
}

// Model
type model struct {
	currentScreen screen
	spinner       spinner.Model
	table         table.Model
	list          list.Model
	menuIndex     int
	backups       []Backup
	databases     []Database
	loading       bool
	err           error
	width         int
	height        int
}

type Backup struct {
	ID           string
	DatabaseName string
	Status       string
	Created      time.Time
	Size         int64
}

type Database struct {
	ID   string
	Name string
	Type string
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create table columns
	columns := []table.Column{
		{Title: "ID", Width: 15},
		{Title: "Database", Width: 20},
		{Title: "Status", Width: 15},
		{Title: "Created", Width: 20},
		{Title: "Size", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Sample data
	backups := []Backup{
		{
			ID:           "backup-001",
			DatabaseName: "production-db",
			Status:       "completed",
			Created:      time.Now().Add(-2 * time.Hour),
			Size:         1024 * 1024 * 512, // 512 MB
		},
		{
			ID:           "backup-002",
			DatabaseName: "staging-db",
			Status:       "running",
			Created:      time.Now().Add(-30 * time.Minute),
			Size:         0,
		},
		{
			ID:           "backup-003",
			DatabaseName: "development-db",
			Status:       "completed",
			Created:      time.Now().Add(-24 * time.Hour),
			Size:         1024 * 1024 * 256, // 256 MB
		},
	}

	databases := []Database{
		{ID: "db-001", Name: "production-db", Type: "PostgreSQL"},
		{ID: "db-002", Name: "staging-db", Type: "MySQL"},
		{ID: "db-003", Name: "development-db", Type: "MongoDB"},
	}

	// Create menu items
	items := []list.Item{
		menuItem{title: "Dashboard", desc: "View backup statistics and recent activity"},
		menuItem{title: "Backups", desc: "View and manage all backups"},
		menuItem{title: "Databases", desc: "Manage database connections"},
		menuItem{title: "Settings", desc: "Configure backup settings"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "DB Backup TUI"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return model{
		currentScreen: screenDashboard,
		spinner:       s,
		table:         t,
		list:          l,
		backups:       backups,
		databases:     databases,
	}
}

// Menu item for list
type menuItem struct {
	title string
	desc  string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Enter):
			if m.currentScreen == screenDashboard {
				selectedItem := m.list.SelectedItem()
				if item, ok := selectedItem.(menuItem); ok {
					switch item.title {
					case "Dashboard":
						m.currentScreen = screenDashboard
					case "Backups":
						m.currentScreen = screenBackups
					case "Databases":
						m.currentScreen = screenDatabases
					case "Settings":
						m.currentScreen = screenSettings
					}
				}
			}

		case key.Matches(msg, keys.Back):
			m.currentScreen = screenDashboard

		case key.Matches(msg, keys.Refresh):
			m.loading = true
			return m, tickCmd()
		}

	case tickMsg:
		m.loading = false
		return m, tickCmd()
	}

	// Update active component
	switch m.currentScreen {
	case screenDashboard:
		m.list, cmd = m.list.Update(msg)
	case screenBackups:
		m.table, cmd = m.table.Update(msg)
	}

	m.spinner, _ = m.spinner.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// Header
	header := titleStyle.Render("🗄️  DB Backup Terminal UI")

	// Content based on screen
	var content string
	switch m.currentScreen {
	case screenDashboard:
		content = m.viewDashboard()
	case screenBackups:
		content = m.viewBackups()
	case screenDatabases:
		content = m.viewDatabases()
	case screenSettings:
		content = m.viewSettings()
	}

	// Status bar
	statusBar := m.renderStatusBar()

	// Help
	help := helpStyle.Render("↑/↓: navigate • enter: select • esc: back • r: refresh • n: new backup • q: quit")

	return docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"\n",
			content,
			"\n",
			help,
			"\n",
			statusBar,
		),
	)
}

func (m model) viewDashboard() string {
	stats := fmt.Sprintf(
		"📊 Statistics:\n"+
			"Total Backups: %d\n"+
			"Databases: %d\n"+
			"Last Backup: %s\n\n",
		len(m.backups),
		len(m.databases),
		m.backups[0].Created.Format("2006-01-02 15:04"),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		successStyle.Render(stats),
		m.list.View(),
	)
}

func (m model) viewBackups() string {
	// Convert backups to table rows
	rows := []table.Row{}
	for _, b := range m.backups {
		rows = append(rows, table.Row{
			b.ID,
			b.DatabaseName,
			b.Status,
			b.Created.Format("2006-01-02 15:04"),
			formatBytes(b.Size),
		})
	}

	m.table.SetRows(rows)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Backups"),
		"\n",
		m.table.View(),
	)
}

func (m model) viewDatabases() string {
	content := titleStyle.Render("Databases") + "\n\n"

	for _, db := range m.databases {
		content += fmt.Sprintf(
			"🗄️  %s (%s)\n   ID: %s\n\n",
			db.Name,
			db.Type,
			db.ID,
		)
	}

	return content
}

func (m model) viewSettings() string {
	content := titleStyle.Render("Settings") + "\n\n"
	content += "⚙️  Configuration\n\n"
	content += "API URL: http://localhost:8080\n"
	content += "Auto-backup: Enabled\n"
	content += "Backup Interval: 24 hours\n"
	content += "Notifications: Enabled\n"

	return content
}

func (m model) renderStatusBar() string {
	status := "Ready"
	if m.loading {
		status = fmt.Sprintf("%s Loading...", m.spinner.View())
	}

	if m.err != nil {
		status = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return statusBarStyle.Render(status)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
