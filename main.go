package main

// -----------------------------------------------------------------------------
// Imports
// -----------------------------------------------------------------------------

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
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
)

var menuOptions = []string{"Projects", "Issues"}

// -----------------------------------------------------------------------------
// Bubble Tables
// -----------------------------------------------------------------------------
func buildColumns(headers []string, rows []table.Row) []table.Column {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	columns := make([]table.Column, len(headers))
	for i, h := range headers {
		columns[i] = table.Column{Title: h, Width: widths[i] + 2}
	}
	return columns
}

func buildProjectItemRows(items []ProjectItem) []table.Row {
	rows := make([]table.Row, len(items))
	for i, item := range items {
		if item.Content.Typename == "DraftIssue" {
			rows[i] = table.Row{"", "", "Draft", item.Content.Title, ""}
			continue
		}
		rows[i] = table.Row{
			fmt.Sprint(item.Content.Number),
			item.Content.State,
			item.Content.Typename,
			item.Content.Title,
			item.Content.Repository.NameWithOwner,
		}
	}
	return rows
}

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
		Typename   string `json:"__typename"`
		Number     int    `json:"number"`
		Title      string `json:"title"`
		State      string `json:"state"`
		URL        string `json:"url"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
	} `json:"content"`
}

type model struct {
	username string
	issues   []Issue
	projects []Project

	screen     screen
	menuCursor int

	issueTable       table.Model
	projectTable     table.Model
	projectItemTable table.Model
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
			if msg.String() == "esc" {
				m.screen = menuScreen
				return m, nil
			}
			var cmd tea.Cmd
			m.issueTable, cmd = m.issueTable.Update(msg)
			return m, cmd
		case projectsScreen:
			if msg.String() == "esc" {
				m.screen = menuScreen
				return m, nil
			}
			if msg.String() == "enter" {
				items, err := fetchProjectItems(m.projects[m.projectTable.Cursor()].ID)
				if err == nil {
					rows := buildProjectItemRows(items)
					m.projectItemTable = table.New(
						table.WithColumns(buildColumns([]string{"#", "State", "Type", "Title", "Repo"}, rows)),
						table.WithRows(rows),
						table.WithFocused(true),
						table.WithHeight(15),
					)
					m.screen = projectItemsScreen
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.projectTable, cmd = m.projectTable.Update(msg)
			return m, cmd
		case projectItemsScreen:
			if msg.String() == "esc" {
				m.screen = projectsScreen
				return m, nil
			}
			var cmd tea.Cmd
			m.projectItemTable, cmd = m.projectItemTable.Update(msg)
			return m, cmd
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
	return titleStyle.Render("Issues (esc to go back, q to quit)") + "\n\n" + m.issueTable.View()
}

// --- Screens View Projects ---
func (m model) viewProjects() string {
	return titleStyle.Render("Projects (esc to go back, q to quit)") + "\n\n" + m.projectTable.View()
}

// --- Screens View Project Items ---
func (m model) viewProjectItems() string {
	return titleStyle.Render("Project Items (esc to go back, q to quit)") + "\n\n" + m.projectItemTable.View()
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
															repository {
																nameWithOwner
															}
													}
													... on PullRequest {
															number
															title
															state
															url
															repository {
																nameWithOwner
															}
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

	issueRows := make([]table.Row, len(issues))
	for i, issue := range issues {
		issueRows[i] = table.Row{fmt.Sprint(issue.Number), issue.State, issue.Title}
	}
	issueTable := table.New(
		table.WithColumns(buildColumns([]string{"#", "State", "Title"}, issueRows)),
		table.WithRows(issueRows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	projects, err := fetchProjects()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching projects:", err)
		os.Exit(1)
	}

	projectRows := make([]table.Row, len(projects))
	for i, project := range projects {
		projectRows[i] = table.Row{fmt.Sprint(project.Number), project.Title, project.ShortDescription}
	}
	projectTable := table.New(
		table.WithColumns(buildColumns([]string{"#", "Title", "Short Description"}, projectRows)),
		table.WithRows(projectRows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	m := model{
		username:     user.Login,
		issues:       issues,
		issueTable:   issueTable,
		projects:     projects,
		projectTable: projectTable,
		screen:       menuScreen,
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
