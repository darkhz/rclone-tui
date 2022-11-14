package ui

import (
	"strings"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// TabDisplay stores a layout to display tabs.
type TabDisplay struct {
	View *tview.TextView

	switcherFunc func(reverse bool) bool
}

var tabDisplay TabDisplay

// SetupTabs sets up the tabs within the display area.
func SetupTabs(tabs string, alignTabView int, switchFunc func(reverse bool) bool, changedFunc func(tab string)) {
	if tabDisplay.View != nil {
		goto FinishSwitcherSetup
	}

	tabDisplay.View = tview.NewTextView()
	tabDisplay.View.SetRegions(true)
	tabDisplay.View.SetDynamicColors(true)
	tabDisplay.View.SetTextAlign(alignTabView)
	tabDisplay.View.SetBackgroundColor(tcell.ColorDefault)
	tabDisplay.View.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		return action, nil
	})

FinishSwitcherSetup:
	tabDisplay.View.SetHighlightedFunc(func(added, removed, remaining []string) {
		if added == nil {
			return
		}

		changedFunc(added[0])
	})

	tabDisplay.switcherFunc = switchFunc
	SetTabs(tabs)
}

// TabView returns the tab display area.
func TabView() *tview.TextView {
	if tabDisplay.View == nil {
		SetupTabs(
			"", tview.AlignCenter,
			func(reverse bool) bool { return false },
			func(tab string) {},
		)
	}

	return tabDisplay.View
}

// SetTabs sets the provided tabs.
func SetTabs(tabs string) {
	tabDisplay.View.SetText(tabs)

	regions := tabDisplay.View.GetRegionIDs()
	if regions != nil {
		tabDisplay.View.Highlight(regions[0])
	}
}

// SelectTab selects a tab.
func SelectTab(tab string) {
	tabDisplay.View.Highlight(tab)
}

// SwitchTabView cycles through the tabs.
func SwitchTabView(reverse bool, view ...*tview.TextView) string {
	var currentView int

	if view == nil && tabDisplay.switcherFunc != nil && !tabDisplay.switcherFunc(reverse) {
		return ""
	}

	textView := tabDisplay.View
	if view != nil {
		textView = view[0]
	}

	regions := textView.GetRegionIDs()
	if len(regions) == 0 {
		return ""
	}

	if highlights := textView.GetHighlights(); highlights != nil {
		for i, region := range regions {
			if region == highlights[0] {
				currentView = i
			}
		}

		if reverse {
			currentView--
		} else {
			currentView++
		}
	}

	if currentView >= len(regions) {
		currentView = 0
	} else if currentView < 0 {
		currentView = len(regions) - 1
	}

	textView.Highlight(regions[currentView])
	textView.ScrollToHighlight()

	return regions[currentView]
}

// HasTab checks if a tab exists.
func HasTab(tab string) bool {
	return strings.Index(tabDisplay.View.GetText(false), tab) != -1
}

// tabEventHandler handles key events for the tab display area.
func tabEventHandler(event *tcell.EventKey) bool {
	if event.Modifiers() != tcell.ModShift {
		return false
	}

	switch event.Key() {
	case tcell.KeyLeft, tcell.KeyRight:
		SwitchTabView(event.Key() == tcell.KeyLeft)
	}

	return true
}
