package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Help provides documentation for a specific
// operation and keybinding.
type Help struct {
	Operation, Keybinding string
}

var helpMap = map[string]map[string][]Help{
	"Application": {
		"Global": {
			{"Open job manager", "Ctrl+j"},
			{"Show view switcher", "Ctrl+n"},
			{"Cancel currently loading job", "Ctrl+x"},
			{"Suspend", "Ctrl+z"},
			{"Quit", "Ctrl+q"},
		},
		"Configuration/Mounts only": {
			{"Select button", "Enter"},
			{"Move between buttons", "Ctrl+Left/Right"},
			{"Move between sections (wizard only)", "Shift+Tab"},
		},
	},
	"Configuration": {
		"Manager": {
			{"Create new", "n"},
			{"Update", "u"},
			{"Delete", "d"},
			{"Filter", "/"},
		},
		"Wizard": {
			{"Jump to option", "Ctrl+f"},
			{"Save", "Ctrl+s"},
			{"Cancel", "Ctrl+c"},
		},
	},
	"Explorer": {
		"General": {
			{"Switch between panes", "Tab"},
			{"Show remotes", "g"},
			{"Filter entries within pane", "/"},
			{"Sort entries within pane", ","},
			{"Navigate between directories", "Left/Right"},
			{"Refresh a pane", "Ctrl+r"},
			{"Cancel fetching remotes", "Ctrl+x"},
		},
		"Item selection": {
			{"Select one item", "Space"},
			{"Inverse selection", "a"},
			{"Select all items", "A"},
			{"Clear selections", "Escape"},
		},
		"Operations": {
			{"Copy selected items", "p"},
			{"Move selected items", "m"},
			{"Delete selected items", "d"},
			{"Make directory", "M"},
			{"Generate public link for item", ";"},
			{"Show remote information", "i"},
		},
	},
	"Mounts": {
		"Manager": {
			{"Create new", "n"},
			{"Unmount", "u"},
			{"Unmount all", "U"},
		},
		"Wizard": {
			{"Create mountpoint", "Ctrl+s"},
			{"Cancel", "Ctrl+c"},
		},
	},
	"Job Manager": {
		"": {
			{"Navigate between jobs", "Down/Up"},
			{"Cancel job", "x"},
			{"Cancel job group", "Ctrl+x"},
		},
	},
}

// ShowHelp shows a modal with documentation.
func ShowHelp() {
	var tabs string
	var keys []string
	var modal *Modal
	var helpView *tview.TextView

	title := tview.NewTextView()
	title.SetText("[::bu]Help")
	title.SetDynamicColors(true)
	title.SetTextAlign(tview.AlignCenter)
	title.SetBackgroundColor(tcell.ColorDefault)

	helpTabView := tview.NewTextView()
	helpTabView.SetRegions(true)
	helpTabView.SetDynamicColors(true)
	helpTabView.SetTextAlign(tview.AlignCenter)
	helpTabView.SetBackgroundColor(tcell.ColorDefault)
	helpTabView.SetHighlightedFunc(func(added, removed, remaining []string) {
		if added == nil {
			return
		}

		for header, contents := range helpMap {
			if strings.ToLower(header) != added[0] {
				continue
			}

			helpView.Clear()

			for subheader, table := range contents {
				fmt.Fprintf(helpView, "[::bu]%s[-:-:-]\n", subheader)

				for _, help := range table {
					fmt.Fprintf(
						helpView, "%s: %s\n",
						help.Operation, help.Keybinding,
					)
				}

				fmt.Fprintf(helpView, "\n\n")
			}

			break
		}
	})

	helpView = tview.NewTextView()
	helpView.SetDynamicColors(true)
	helpView.SetBackgroundColor(tcell.ColorDefault)
	helpView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			modal.Exit()

		case tcell.KeyTab, tcell.KeyBacktab:
			SwitchTabView(event.Key() == tcell.KeyBacktab, helpTabView)
		}

		return event
	})

	navigateHelp := tview.NewTextView()
	navigateHelp.SetDynamicColors(true)
	navigateHelp.SetBackgroundColor(tcell.ColorDefault)

	for header := range helpMap {
		keys = append(keys, header)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, tab := range keys {
		tabs += fmt.Sprintf(
			"[aqua::b][\"%s\"]%s[\"\"][-:-:-] ",
			strings.ToLower(tab), tab,
		)
	}

	helpTabView.SetText(tabs)
	helpTabView.Highlight(strings.ToLower(keys[0]))

	navigateHelp.SetText("Press [::b]Tab[-:-:-] to switch sections, [::b]Arrow keys[-:-:-] to scroll.")

	helpFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 2, 0, false).
		AddItem(helpTabView, 1, 0, false).
		AddItem(helpView, 0, 1, true).
		AddItem(navigateHelp, 1, 0, false)

	modal = NewCustomModal("help", helpFlex, 60, 100)
	modal.Show()
}
