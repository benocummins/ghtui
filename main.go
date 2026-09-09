package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/api"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	openStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	closedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

var menuOptions = []string{"Projects", "Issues"}

type screen int

const (
	menuScreen screen = iota
	issuesScreen
)

type User struct {
	Login string `json:"login"`
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

type model struct {
	username string
	issues   []Issue

	screen      screen
	menuCursor  int
	issueCursor int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		switch m.screen {
		case menuScreen:
			switch msg.String() {
			case "up", "k":
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case "down", "j":
				if m.menuCursor < len(menuOptions)-1 {
					m.menuCursor++
				}
			case "enter":
				if menuOptions[m.menuCursor] == "Issues" {
					m.screen = issuesScreen
				}
				// "Projects" not wired up yet - see issue = #9
			}
		case issuesScreen:
			switch msg.String() {
			case "esc":
				m.screen = menuScreen
			case "up", "k":
				if m.issueCursor > 0 {
					m.issueCursor--
				}
			case "down", "j":
				if m.issueCursor < len(m.issues)-1 {
					m.issueCursor++
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case issuesScreen:
		return m.viewIssues()
	default:
		return m.viewMenu()
	}
}

func (m model) viewMenu() string {
	s := titleStyle.Render(fmt.Sprintf("Hi, %s! What do you want to review today?", m.username)) + "\n\n"
	for i, opt := range menuOptions {
		cursor := "  "
		line := opt
		if m.menuCursor == i {
			cursor = "> "
			line = selectedStyle.Render(line)
		}
		s += cursor + line + "\n"
	}
	s += "\n(enter to select, q to quit)"
	return s
}

func (m model) viewIssues() string {
	s := titleStyle.Render("Issues (esc to go back, q to quit)") + "\n\n"
	for i, issue := range m.issues {
		stateStyle := openStyle
		if issue.State == "closed" {
			stateStyle = closedStyle
		}
		line := fmt.Sprintf("#%d [%s] %s", issue.Number, stateStyle.Render(issue.State), issue.Title)

		cursor := "  "
		if m.issueCursor == i {
			cursor = "> "
			line = selectedStyle.Render(line)
		}
		s += cursor + line + "\n"
	}
	return s
}

func fetchUser() (User, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return User{}, err
	}
	var user User
	err = client.Get("user", &user)
	return user, err
}

func fetchIssues(repo string) ([]Issue, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	var issues []Issue
	err = client.Get(fmt.Sprintf("repos/%s/issues", repo), &issues)
	return issues, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ghtui <owner>/<repo>")
		os.Exit(1)
	}

	user, err := fetchUser()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching user:", err)
		os.Exit(1)
	}

	issues, err := fetchIssues(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching issues:", err)
		os.Exit(1)
	}

	m := model{
		username: user.Login,
		issues:   issues,
		screen:   menuScreen,
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
