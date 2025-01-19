# mrun

<p>
  <a href="https://pkg.go.dev/github.com/zmwangx/mrun"><img src="https://pkg.go.dev/badge/github.com/zmwangx/mrun.svg" alt="Go Reference"></a>
</p>

Embedded TUI multi-command runner based on bubbletea.

![Demo](https://github.com/user-attachments/assets/936f906a-d069-4d04-810a-fc7812e29644)

Sometimes you want to run multiple commands in parallel and let the user monitor their progress in real time. tmux is an obvious option but it's highly unusual to spawn tmux session/windows as a third party program, and you can't control spawned commands as effectively when they're children of tmux. `mrun` lets you integrate a TUI grid right inside your program, neatly solving the problem.

See [demo.go](cmd/demo/demo.go) for the code used to generate the demo image.

## Features

Supports:

- Things you expect from a TUI built in the 2020s: Unicode support (CJK-aware), color.
- Commands are run in ptys, don't need to mess with flags to reenable interactive features.
- Each command has a scroll buffer.
- Carriage returns are handled gracefully, so commands with basic progress bars work as expected.
- Mouse support: click to focus, mouse wheel to scroll.
- Terminal resizing is handled gracefully.

Does not support:

- Input: commands can't receive user input.
- Terminal emulation: if your command itself is a TUI program using escape sequences to write to specific coordinates on screen, then it won't work correctly and will likely mess up everything. The output buffer implementation of `mrun` is basic, far from a full blown terminal emulator like tmux. Try to use the batch mode of your program if it has one (e.g. by piping its output to cat).
- Integration into larger bubbletea applications: currently not exposed.

## Documentation

See <https://pkg.go.dev/github.com/zmwangx/mrun>.

## Controls

- Focusing pane: tab for next pane, shift+tab for previous pane, click to focus any pane.
- Scrolling inside pane: up, down, page up, page down, mouse wheel.
- Manual interrupt: ctrl+c, esc, q.
- Dialog: tab/shift+tab/left/right to navigate between buttons, enter to confirm, esc/q to cancel.
