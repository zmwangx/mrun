package mrun

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/reflow/wrap"
)

var (
	_activePaneBorderColor   = lipgloss.Color("168") // HotPink3
	_inactivePaneBorderColor = lipgloss.Color("241") // Grey39
	_commandColor            = lipgloss.Color("75")  // SteelBlue1
	_errorColor              = lipgloss.Color("196") // Red1

	_paneStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(false)
	_activePaneStyle      = _paneStyle.BorderForeground(_activePaneBorderColor)
	_inactivePaneStyle    = _paneStyle.BorderForeground(_inactivePaneBorderColor)
	_activeOverlayStyle   = lipgloss.NewStyle().Foreground(_activePaneBorderColor)
	_inactiveOverlayStyle = lipgloss.NewStyle().Foreground(_inactivePaneBorderColor)
	_commandStyle         = lipgloss.NewStyle().
				Foreground(_commandColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(_commandColor).
				BorderBottom(true)
	_errorStyle = lipgloss.NewStyle().Foreground(_errorColor)
)

type model struct {
	id    string
	ready bool

	executor    *multiExecutor
	count       int
	rows        int
	cols        int
	panes       []modelPane
	activePane  int
	allDone     bool
	terminating bool

	dialogActive bool
	dialog       dialogModel
	autoQuit     bool
}

type modelPane struct {
	cmd              *Command
	printCommandLine bool
	label            string
	title            string
	content          string
	// Store the last line of the content separately if it's not terminated by
	// an LF so that we can support overwriting with CR.
	lastLine string
	// Viewport width and height.
	vw, vh int
	v      viewport.Model
	// The winsizeCh channel is used to send viewport size changes to the
	// command executor. It's returned by runCommand().
	winsizeCh chan<- winsize
	exited    bool
	exitCode  int
	errored   bool
	err       error
}

type winsize struct {
	w, h int
}

type (
	exitDialogOpenMsg struct{}
	dialogCloseMsg    struct{}
	terminateMsg      struct{}
)

// cols must be positive, commands must not be empty.
func newModel(cols int, commands []*Command, opts runOpts) model {
	count := len(commands)
	rows := (count + cols - 1) / cols
	var panes []modelPane
	for _, c := range commands {
		pane := modelPane{
			cmd:              c,
			printCommandLine: opts.printCommandLine,
			label:            c.label,
		}
		var title string
		if len(pane.cmd.cmd.Args) > 0 {
			title = pane.cmd.cmd.Args[0]
			if c.label != "" && title != c.label {
				title = fmt.Sprintf("%s (%s)", title, c.label)
			}
		} else {
			title = c.label
		}
		pane.title = title
		panes = append(panes, pane)
	}
	return model{
		id:       zone.NewPrefix(),
		executor: newMultiExecutor(),
		count:    count,
		rows:     rows,
		cols:     cols,
		panes:    panes,
		dialog:   newDialogModel(),
		autoQuit: opts.autoQuit,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	addCmd := func(c tea.Cmd) { cmds = append(cmds, c) }
	// Shortcut for returning the model and a batch of commands.
	ret := func() (model, tea.Cmd) { return m, tea.Batch(cmds...) }

	setWindowTitle := func() {
		addCmd(m.setWindowTitleToActivePane())
	}
	setActivePane := func(idx int) {
		m.activePane = idx
		setWindowTitle()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width / m.cols
		wRem := msg.Width % m.cols
		h := msg.Height / m.rows
		hRem := msg.Height % m.rows
		for row := 0; row < m.rows; row++ {
			for col := 0; col < m.cols; col++ {
				idx := row*m.cols + col
				if idx >= m.count {
					break
				}
				vw := w - 1
				if col < wRem {
					vw++
				}
				vh := h - 1
				if row < hRem {
					vh++
				}
				pane := &m.panes[idx]
				if !m.ready {
					// Run command on first WindowSizeMsg.
					pane.winsizeCh, cmd = m.executor.runCommand(idx, vw, vh, pane.cmd)
					addCmd(cmd)
				} else {
					// Resize command pty on subsequent WindowSizeMsgs.
					if !pane.exited && !pane.errored {
						go func() {
							pane.winsizeCh <- winsize{vw, vh}
						}()
					}
				}
				pane.vw = vw
				pane.vh = vh
				pane.v = viewport.New(vw, vh)
				pane.refreshContent()
				pane.v.GotoBottom()
			}
		}
		if !m.ready {
			setWindowTitle()
			addCmd(m.executor.waitForAllDone)
			m.ready = true
		}
		return ret()

	case tea.KeyMsg:
		if m.blocked() {
			m.dialog, cmd = m.dialog.Update(msg)
			addCmd(cmd)
			break
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			addCmd(openExitDialog())
			return ret()
		case "tab":
			setActivePane((m.activePane + 1) % m.count)
			return ret()
		case "shift+tab":
			setActivePane((m.activePane - 1 + m.count) % m.count)
			return ret()
		}

	case tea.MouseMsg:
		if m.blocked() {
			break
		}
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			break
		}
		for idx := 0; idx < m.count; idx++ {
			if zone.Get(m.paneId(idx)).InBounds(msg) {
				setActivePane(idx)
				return ret()
			}
		}

	case cmdOutputMsg:
		addCmd(func() tea.Msg {
			return m.executor.nextOutput(msg)
		})

		line := string(msg.line)
		if len(line) == 0 {
			break
		}
		pane := &m.panes[msg.paneIdx]
		switch line[len(line)-1] {
		case '\n':
			pane.lastLine = ""
			pane.content += line
		case '\r':
			pane.lastLine = line[:len(line)-1]
		default:
			// This shouldn't happen, but just in case.
			pane.lastLine += line
		}
		atBottom := pane.v.AtBottom()
		pane.refreshContent()
		// Only auto-scroll if the viewport was already at the bottom.
		if atBottom {
			pane.v.GotoBottom()
		}
		return ret()

	case cmdExitMsg:
		pane := &m.panes[msg.paneIdx]
		pane.exited = msg.exited
		pane.exitCode = msg.exitCode
		pane.errored = msg.errored
		pane.err = msg.err
		return ret()

	case allDoneMsg:
		if m.autoQuit {
			return m, tea.Quit
		}
		m.allDone = true
		addCmd(openExitDialog())
		return ret()

	case exitDialogOpenMsg:
		m.dialogActive = true
		m.dialog.reset()
		m.dialog.prompt = "Interrupt? All running commands will be gracefully terminated."
		if m.allDone {
			m.dialog.prompt = "All done. Quit?"
		}
		m.dialog.buttons = []dialogButton{
			{"Yes", terminate()},
			{"No", closeDialog()},
		}
		m.dialog.selected = 0
		return ret()

	case dialogCloseMsg:
		m.dialogActive = false
		return ret()

	case terminateMsg:
		m.dialogActive = false
		m.terminating = true
		addCmd(m.executor.terminateAll)
		return ret()

	case allTerminatedMsg:
		return m, tea.Quit
	}

	if !m.dialogActive {
		// Handle keyboard and mouse events in the viewport.
		m.panes[m.activePane].v, cmd = m.panes[m.activePane].v.Update(msg)
		addCmd(cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return ""
	}
	var rowBlocks []string
	for row := 0; row < m.rows; row++ {
		var blocks []string
		for col := 0; col < m.cols; col++ {
			idx := row*m.cols + col
			if idx >= m.count {
				break
			}
			pane := m.panes[idx]
			v := pane.v
			isActive := idx == m.activePane

			var style lipgloss.Style
			if isActive {
				style = _activePaneStyle
			} else {
				style = _inactivePaneStyle
			}
			block := style.Render(v.View())
			w, h := lipgloss.Size(block)

			styleOverlay := func(s string) string {
				if isActive {
					return _activeOverlayStyle.Render(s)
				}
				return _inactiveOverlayStyle.Render(s)
			}

			// Overlay label in bottom center.
			if pane.label != "" {
				label := pane.label
				if len(label) > w-2 {
					label = label[:w-2]
				}
				labelOverlay := styleOverlay(" " + label + " ")
				block = placeOverlay((w-lipgloss.Width(labelOverlay))/2, h-1, labelOverlay, block)
			}

			// Overlay exit status or error in the bottom left corner.
			var exitOverlay string
			if pane.errored {
				exitOverlay = _errorStyle.Render("ERROR ")
			} else if pane.exited {
				code := pane.exitCode
				s := fmt.Sprintf("EXIT %d ", code)
				if code == 0 {
					exitOverlay = styleOverlay(s)
				} else {
					exitOverlay = _errorStyle.Render(s)
				}
			}
			block = placeOverlay(0, h-1, exitOverlay, block)

			// Overlay scroll percentage in bottom right corner.
			scrollOverlay := styleOverlay(fmt.Sprintf(" %.0f%% ", v.ScrollPercent()*100))
			block = placeOverlay(w-lipgloss.Width(scrollOverlay)-1, h-1, scrollOverlay, block)

			blocks = append(blocks, zone.Mark(m.paneId(idx), block))
		}
		rowBlocks = append(rowBlocks, lipgloss.JoinHorizontal(lipgloss.Top, blocks...))
	}
	view := lipgloss.JoinVertical(lipgloss.Left, rowBlocks...)
	vw, vh := lipgloss.Size(view)

	// Render dialog.
	var dialogView string
	if m.dialogActive {
		dialogView = m.dialog.View()
	}
	if m.terminating {
		dialogView = dialogFeedbackView("Terminating...")
	}
	if dialogView != "" {
		dw, dh := lipgloss.Size(dialogView)
		dx := (vw - dw) / 2
		if dx < 0 {
			dx = 0
		}
		dy := (vh - dh) / 2
		if dy < 0 {
			dy = 0
		}
		// TODO: support mouse events in the dialog.
		//
		// The PlaceOverlay() code seems overly simplistic and destroys zone markers
		// in the overlay and may or may not fuck up markers elsewhere, so we
		// probably have to wait for proper compositing support.
		//
		// See
		// - https://github.com/charmbracelet/lipgloss/issues/65
		// - https://github.com/charmbracelet/lipgloss/pull/102
		// - https://github.com/charmbracelet/bubbletea/issues/79
		view = placeOverlay(dx, dy, dialogView, view)
	}

	return zone.Scan(view)
}

func (m model) blocked() bool {
	return m.dialogActive || m.terminating
}

func (m model) setWindowTitleToActivePane() tea.Cmd {
	return tea.SetWindowTitle(m.panes[m.activePane].title)
}

func (m model) finalView() string {
	m.dialogActive = false
	m.terminating = false
	return m.View()
}

func (p *modelPane) refreshContent() {
	var header string
	if p.printCommandLine {
		header = _commandStyle.Width(p.vw).Render(p.cmd.cmdline)
	}
	content := wrap.String(p.content+p.lastLine, p.vw)
	p.v.SetContent(lipgloss.JoinVertical(lipgloss.Left, header, content))
}

func openExitDialog() tea.Cmd {
	return func() tea.Msg {
		return exitDialogOpenMsg{}
	}
}

func closeDialog() tea.Cmd {
	return func() tea.Msg {
		return dialogCloseMsg{}
	}
}

func terminate() tea.Cmd {
	return func() tea.Msg {
		return terminateMsg{}
	}
}

func (m model) paneId(idx int) string {
	return m.id + fmt.Sprintf("pane%d", idx)
}
