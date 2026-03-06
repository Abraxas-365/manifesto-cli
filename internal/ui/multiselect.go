package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// SelectableItem represents an item in the multi-select menu.
type SelectableItem struct {
	Name        string
	Description string
	Selected    bool
}

// MultiSelect displays an interactive checkbox menu and returns selected items.
// Navigation: up/down arrows, space to toggle, 'a' to toggle all, enter to confirm.
func MultiSelect(title string, items []SelectableItem) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// Switch terminal to raw mode.
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Non-interactive: skip selection.
		return nil, nil
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, nil // Fallback: skip selection if raw mode fails.
	}
	defer term.Restore(fd, oldState)

	cursor := 0

	render := func() {
		// Move to start and clear.
		var buf strings.Builder
		// Move cursor up to overwrite previous render (except first time).
		buf.WriteString("\r")

		buf.WriteString("  " + title + "\r\n")
		buf.WriteString(Dim.Sprint("  ↑/↓ navigate  ⎵ toggle  a all  enter confirm") + "\r\n")
		buf.WriteString("\r\n")

		for i, item := range items {
			check := "○"
			if item.Selected {
				check = Green.Sprint("●")
			}

			if i == cursor {
				buf.WriteString(fmt.Sprintf("  %s %s  %-8s  %s\r\n",
					Cyan.Sprint("❯"),
					check,
					Bold.Sprint(item.Name),
					Dim.Sprint(item.Description),
				))
			} else {
				buf.WriteString(fmt.Sprintf("    %s  %-8s  %s\r\n",
					check,
					item.Name,
					Dim.Sprint(item.Description),
				))
			}
		}

		selectedNames := selectedItemNames(items)
		if len(selectedNames) > 0 {
			buf.WriteString("\r\n")
			buf.WriteString(fmt.Sprintf("  %s %s\r\n",
				Green.Sprint(fmt.Sprintf("%d selected:", len(selectedNames))),
				strings.Join(selectedNames, ", "),
			))
		} else {
			buf.WriteString("\r\n")
			buf.WriteString(Dim.Sprint("  No modules selected (press enter to skip)") + "\r\n")
		}

		fmt.Print(buf.String())
	}

	clearRender := func() {
		// lines = title(1) + help(1) + blank(1) + items(len) + blank(1) + selected(1)
		totalLines := 3 + len(items) + 2
		for i := 0; i < totalLines; i++ {
			fmt.Print("\033[2K") // Clear line
			if i < totalLines-1 {
				fmt.Print("\033[A") // Move up
			}
		}
		fmt.Print("\r")
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}

		// Parse input.
		if n == 1 {
			switch buf[0] {
			case 13: // Enter
				clearRender()
				selected := selectedItemNames(items)
				if len(selected) > 0 {
					fmt.Printf("  %s Modules: %s\r\n",
						Green.Sprint("✓"),
						strings.Join(selected, ", "),
					)
				} else {
					fmt.Printf("  %s No modules selected\r\n", Dim.Sprint("○"))
				}
				fmt.Println()
				return selected, nil

			case ' ': // Space - toggle
				items[cursor].Selected = !items[cursor].Selected
				clearRender()
				render()

			case 'a', 'A': // Toggle all
				allSelected := true
				for _, item := range items {
					if !item.Selected {
						allSelected = false
						break
					}
				}
				for i := range items {
					items[i].Selected = !allSelected
				}
				clearRender()
				render()

			case 3: // Ctrl+C
				clearRender()
				return nil, fmt.Errorf("interrupted")

			case 'k': // vim up
				if cursor > 0 {
					cursor--
				}
				clearRender()
				render()

			case 'j': // vim down
				if cursor < len(items)-1 {
					cursor++
				}
				clearRender()
				render()
			}
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if cursor > 0 {
					cursor--
				}
				clearRender()
				render()
			case 66: // Down arrow
				if cursor < len(items)-1 {
					cursor++
				}
				clearRender()
				render()
			}
		}
	}
}

func selectedItemNames(items []SelectableItem) []string {
	var names []string
	for _, item := range items {
		if item.Selected {
			names = append(names, item.Name)
		}
	}
	return names
}
