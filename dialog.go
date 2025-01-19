package mrun

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	_dialogWidth = 40

	_dialogPromptColor     = lipgloss.Color("255") // Grey93
	_dialogBgColor         = lipgloss.Color("56")  // Purple3
	_buttonTextColor       = lipgloss.Color("255") // Grey93
	_activeButtonBgColor   = lipgloss.Color("168") // HotPink3
	_inactiveButtonBgColor = lipgloss.Color("246") // Grey58

	_dialogBoxStyle    = lipgloss.NewStyle().Padding(1).Background(_dialogBgColor)
	_dialogPromptStyle = lipgloss.NewStyle().
				Width(_dialogWidth - 2).Align(lipgloss.Center).
				Foreground(_dialogPromptColor).Background(_dialogBgColor)

	_buttonStyle         = lipgloss.NewStyle().Padding(0, 1).Foreground(_buttonTextColor)
	_activeButtonStyle   = _buttonStyle.Background(_activeButtonBgColor)
	_inactiveButtonStyle = _buttonStyle.Background(_inactiveButtonBgColor)
	_buttonSpacerStyle   = lipgloss.NewStyle().Background(_dialogBgColor)
)

type dialogButton struct {
	text string
	cmd  tea.Cmd
}

type dialogModel struct {
	prompt   string
	buttons  []dialogButton
	selected int
}

func newDialogModel() dialogModel {
	return dialogModel{}
}

func (m dialogModel) Init() tea.Cmd {
	return nil
}

func (m dialogModel) Update(msg tea.Msg) (dialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.selected = (m.selected + 1) % len(m.buttons)
		case "shift+tab":
			m.selected = (m.selected - 1 + len(m.buttons)) % len(m.buttons)
		case "right":
			if m.selected < len(m.buttons)-1 {
				m.selected++
			}
		case "left":
			if m.selected > 0 {
				m.selected--
			}
		case "enter":
			return m, m.buttons[m.selected].cmd
		case "q", "esc":
			return m, closeDialog()
		}
	}
	return m, nil
}

func (m dialogModel) View() string {
	if len(m.buttons) == 0 {
		return ""
	}
	dialog := _dialogPromptStyle.Render(m.prompt)
	var renderedButtons []string
	for i, b := range m.buttons {
		style := _inactiveButtonStyle
		if i == m.selected {
			style = _activeButtonStyle
		}
		renderedButtons = append(renderedButtons, style.Render(b.text))
		// Add a styled spacer; using margin will result in a spacer without proper background.
		if i < len(m.buttons)-1 {
			renderedButtons = append(renderedButtons, _buttonSpacerStyle.Render(" "))
		}
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, renderedButtons...)
	// Use PlaceHorizontal to make sure the entire button row is centered and
	// has proper background.
	buttons = lipgloss.PlaceHorizontal(_dialogWidth-2, lipgloss.Center, buttons, lipgloss.WithWhitespaceBackground(_dialogBgColor))
	return _dialogBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Center, dialog, "", buttons))
}

func (m *dialogModel) reset() {
	m.prompt = ""
	m.buttons = nil
	m.selected = 0
}

func dialogFeedbackView(text string) string {
	return _dialogBoxStyle.Render(_dialogPromptStyle.Render(text))
}
