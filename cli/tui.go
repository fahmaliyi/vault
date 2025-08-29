package cli

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fahmaliyi/vault/vault"
	"github.com/google/uuid"
)

type model struct {
	vault      *vault.Vault
	entries    []vault.Entry
	cursor     int
	state      string // "table", "showEntry", "addEntry"
	textInputs []textinput.Model
	selected   *vault.Entry
	clipboard  string
	msg        string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Underline(true)
	msgStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("0"))
)

// RunTUI starts the interactive TUI
func RunTUI(v *vault.Vault) {
	m := model{
		vault:   v,
		entries: v.List(),
		state:   "table",
	}

	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		fmt.Println("Error starting TUI:", err)
	}
}

// --- Tea Model interface ---
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case "table":
		return updateTable(m, msg)
	case "showEntry":
		return updateShowEntry(m, msg)
	case "addEntry":
		return updateAddEntry(m, msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	switch m.state {
	case "table":
		return viewTable(m)
	case "showEntry":
		return viewShowEntry(m)
	case "addEntry":
		return viewAddEntry(m)
	default:
		return "Unknown state"
	}
}

// --- Table ---
func updateTable(m model, msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			m.selected = &m.entries[m.cursor]
			m.state = "showEntry"
		case "a":
			// Call CLI add entry while staying in TUI
			AddEntryCLI(m.vault)
			// Refresh entries after adding
			m.entries = m.vault.List()

		case "d":
			e := m.entries[m.cursor]
			m.vault.Delete(e.ID)
			m.vault.Save()
			m.entries = m.vault.List()
			if m.cursor >= len(m.entries) && m.cursor > 0 {
				m.cursor--
			}
		case "c":
			e := m.entries[m.cursor]
			clipboard.WriteAll(string(e.Secret))
			m.msg = "Password copied! (clears in 30s)"
			go func() {
				time.Sleep(30 * time.Second)
				clipboard.WriteAll("")
			}()
		}
	}
	return m, nil
}

func viewTable(m model) string {
	s := titleStyle.Render("Vault Entries") + "\n\n"
	for i, e := range m.entries {
		line := fmt.Sprintf("%-36s  %-20s  %-20s", e.ID, e.Title, e.Username)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		s += line + "\n"
	}
	if m.msg != "" {
		s += "\n" + msgStyle.Render(m.msg)
	}
	s += "\nCommands: j/k=move, enter=show, a=add, d=delete, c=copy, q=quit"
	return s
}

// --- Show Entry ---
func updateShowEntry(m model, msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = "table"
			m.selected = nil
		case "v":
			// temporarily reveal secret
			m.msg = fmt.Sprintf("Secret: %s", string(m.selected.Secret))
			go func() {
				time.Sleep(5 * time.Second)
				m.msg = ""
			}()
		}
	}
	return m, nil
}

func viewShowEntry(m model) string {
	e := m.selected
	s := fmt.Sprintf("Title: %s\nUsername: %s\nNotes: %s\nSecret: %s\n",
		e.Title, e.Username, e.Notes, "********")
	if m.msg != "" {
		s += "\n" + msgStyle.Render(m.msg)
	}
	s += "\nPress 'v' to reveal, Esc to return"
	return s
}

// --- Add Entry ---
func updateAddEntry(m model, msg tea.Msg) (model, tea.Cmd) {
	// Update the focused text input
	for i := range m.textInputs {
		ti := &m.textInputs[i]
		if ti.Focused() {
			var _ tea.Cmd
			*ti, _ = ti.Update(msg)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "down":
			// Move focus to next input
			m.focusNext(msg.String() == "shift+tab")
		case "esc":
			m.state = "table"
		case "ctrl+s": // optional: explicit save
			if allInputsFilled(m.textInputs) {
				m = saveAddEntry(m)
			}
		case "enter":
			// If last input is focused and all filled, save
			if m.textInputs[len(m.textInputs)-1].Focused() && allInputsFilled(m.textInputs) {
				m = saveAddEntry(m)
			}
		}
	}

	return m, nil
}

// Focus next or previous input
func (m *model) focusNext(backward bool) {
	n := len(m.textInputs)
	for i := 0; i < n; i++ {
		if m.textInputs[i].Focused() {
			m.textInputs[i].Blur()
			if backward {
				m.textInputs[(i-1+n)%n].Focus()
			} else {
				m.textInputs[(i+1)%n].Focus()
			}
			break
		}
	}
}

// Save the entry to vault
func saveAddEntry(m model) model {
	e := vault.Entry{
		ID:       uuid.New().String(),
		Title:    m.textInputs[0].Value(),
		Username: m.textInputs[1].Value(),
		Secret:   []byte(m.textInputs[2].Value()),
		Notes:    m.textInputs[3].Value(),
	}
	m.vault.Add(e)
	m.vault.Save()
	m.entries = m.vault.List()
	m.state = "table"

	// Clear text inputs
	for i := range m.textInputs {
		m.textInputs[i].SetValue("")
	}
	return m
}

func viewAddEntry(m model) string {
	s := titleStyle.Render("Add New Entry") + "\n\n"
	for i, ti := range m.textInputs {
		s += fmt.Sprintf("%s: %s\n", ti.Placeholder, ti.View())
		if i < len(m.textInputs)-1 {
			s += "\n"
		}
	}
	s += "\nPress Enter to save, Esc to cancel"
	return s
}

func allInputsFilled(inputs []textinput.Model) bool {
	for _, ti := range inputs {
		if ti.Value() == "" {
			return false
		}
	}
	return true
}
