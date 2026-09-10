package main

// -----------------------------------------------------------------------------
// Imports
// -----------------------------------------------------------------------------

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/api"
)

// -----------------------------------------------------------------------------
// Styling
// -----------------------------------------------------------------------------

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	openStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	closedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

var menuOptions = []string{"Projects", "Issues"}

// -----------------------------------------------------------------------------
// Data Structures
// -----------------------------------------------------------------------------

type User struct {
	Login string `json:"login"`
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

type Project struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Number           int    `json:"number"`
	ShortDescription string `json:"shortDescription"`
	URL              string `json:"url"`
}

type ProjectItem struct {
	ID      string `json:"id"`
	Content struct {
		Typename string `json:"__typename"`
		Number   int    `json:"number"`
		Title    string `json:"title"`
		State    string `json:"state"`
		URL      string `json:"url"`
	} `json:"content"`
}

type model struct {
	username     string
	issues       []Issue
	projects     []Project
	projectItems []ProjectItem

	screen            screen
	menuCursor        int
	issueCursor       int
	projectCursor     int
	projectItemCursor int
}

// -----------------------------------------------------------------------------
// Screens
// -----------------------------------------------------------------------------

type screen int

const (
	menuScreen screen = iota
	issuesScreen
	projectsScreen
	projectItemsScreen
)

func (m model) Init() tea.Cmd {
	return nil
}

// --- Screens Update ---
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
				switch menuOptions[m.menuCursor] {
				case "Issues":
					m.screen = issuesScreen
				case "Projects":
					m.screen = projectsScreen
				}
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
		case projectsScreen:
			switch msg.String() {
			case "esc":
				m.screen = menuScreen
			case "up", "k":
				if m.projectCursor > 0 {
					m.projectCursor--
				}
			case "down", "j":
				if m.projectCursor < len(m.projects)-1 {
					m.projectCursor++
				}
			case "enter":
				items, err := fetchProjectItems(m.projects[m.projectCursor].ID)
				if err == nil {
					m.projectItems = items
					m.projectItemCursor = 0
					m.screen = projectItemsScreen
				}
			}
		case projectItemsScreen:
			switch msg.String() {
			case "esc":
				m.screen = projectsScreen
			case "up", "k":
				if m.projectItemCursor > 0 {
					m.projectItemCursor--
				}
			case "down", "j":
				if m.projectItemCursor < len(m.projectItems)-1 {
					m.projectItemCursor++
				}
			}
		}
	}
	return m, nil
}

// --- Screens View ---
func (m model) View() string {
	switch m.screen {
	case issuesScreen:
		return m.viewIssues()
	case projectsScreen:
		return m.viewProjects()
	case projectItemsScreen:
		return m.viewProjectItems()
	default:
		return m.viewMenu()
	}
}

// -- Screens View Menu ---
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

// --- Screens View Issues ---
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

// --- Screens View Projects ---
func (m model) viewProjects() string {
	s := titleStyle.Render("Projects (esc to go back, q to quit)") + "\n\n"
	for i, project := range m.projects {
		line := fmt.Sprintf("#%d %s", project.Number, project.Title)
		if project.ShortDescription != "" {
			line += " — " + project.ShortDescription
		}

		cursor := "  "
		if m.projectCursor == i {
			cursor = "> "
			line = selectedStyle.Render(line)
		}
		s += cursor + line + "\n"
	}
	return s
}

// --- Screens View Project Items ---
func (m model) viewProjectItems() string {
	s := titleStyle.Render("Project Items (esc to go back, q to quit)") + "\n\n"
	for i, projectItem := range m.projectItems {
		if projectItem.Content.Typename == "DraftIssue" {
			line := fmt.Sprintf("-- [DRAFT] %s %s", projectItem.Content.Typename, projectItem.Content.Title)

			cursor := "  "
			if m.projectItemCursor == i {
				cursor = "> "
				line = selectedStyle.Render(line)
			}
			s += cursor + line + "\n"
		} else {
			stateStyle := openStyle
			if projectItem.Content.State == "closed" {
				stateStyle = closedStyle
			}
			line := fmt.Sprintf("#%d [%s] %s %s", projectItem.Content.Number, stateStyle.Render(projectItem.Content.State), projectItem.Content.Typename, projectItem.Content.Title)

			cursor := "  "
			if m.projectItemCursor == i {
				cursor = "> "
				line = selectedStyle.Render(line)
			}
			s += cursor + line + "\n"
		}
	}
	return s
}

// -----------------------------------------------------------------------------
// Fetch Data
// -----------------------------------------------------------------------------

// --- Fetch User Data ---
func fetchUser() (User, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return User{}, err
	}
	var user User
	err = client.Get("user", &user)
	return user, err
}

// --- Fetch Issue Data ---
func fetchIssues(repo string) ([]Issue, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	var issues []Issue
	err = client.Get(fmt.Sprintf("repos/%s/issues", repo), &issues)
	return issues, err
}

// -- Fetch Project Data ---
func fetchProjects() ([]Project, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	query := `
	query {
			viewer {
					projectsV2(first:20) {
						nodes {
								id
								title
								number
								shortDescription
								url
						}
					}
			}
	}`

	var resp struct {
		Viewer struct {
			ProjectsV2 struct {
				Nodes []Project `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"viewer"`
	}

	err = client.Do(query, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Viewer.ProjectsV2.Nodes, nil

}

// --- Fetch Project Items ---
func fetchProjectItems(projectID string) ([]ProjectItem, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}

	query := `
	query($projectId: ID!) {
			node(id: $projectId) {
					... on ProjectV2 {
							items(first: 50) {
									nodes {
											id
											content {
													__typename
													... on Issue {
															number
															title
															state
															url
													}
													... on PullRequest {
															number
															title
															state
															url
													}
													... on DraftIssue {
															title
													}
											}
									}
							}
					}
			}
	}`

	variables := map[string]interface{}{
		"projectId": projectID,
	}

	var resp struct {
		Node struct {
			Items struct {
				Nodes []ProjectItem `json:"nodes"`
			} `json:"items"`
		} `json:"node"`
	}

	err = client.Do(query, variables, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Node.Items.Nodes, nil
}

// -----------------------------------------------------------------------------
// Main
// -----------------------------------------------------------------------------

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

	projects, err := fetchProjects()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching projects:", err)
		os.Exit(1)
	}

	m := model{
		username: user.Login,
		issues:   issues,
		projects: projects,
		screen:   menuScreen,
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
