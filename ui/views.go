package ui

import (
	"strconv"
	"strings"

	"github.com/darkhz/rclone-tui/cmd"
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// View describes the layout for a view page.
type View interface {
	Init() bool

	Exit(page string) bool

	Name() string

	Focused() string

	Layout() tview.Primitive
}

// ViewBar stores a layout for the title bar.
type ViewBar struct {
	Title *tview.TextView

	Flex          *tview.Flex
	ConnIndicator *tview.TextView
	JobIndicator  *tview.TextView
}

var (
	viewBar ViewBar

	currentView View
	views       = []View{&dashboard, &configuration, &explorer, &mounts}
)

// ViewTitle returns the title bar.
func ViewTitle() tview.Primitive {
	viewBar.Title = tview.NewTextView()
	viewBar.Title.SetDynamicColors(true)
	viewBar.Title.SetBackgroundColor(tcell.ColorDefault)

	viewBar.ConnIndicator = tview.NewTextView()
	viewBar.ConnIndicator.SetDynamicColors(true)
	viewBar.ConnIndicator.SetText("Disconnected")
	viewBar.ConnIndicator.SetTextAlign(tview.AlignCenter)
	viewBar.ConnIndicator.SetBackgroundColor(tcell.ColorDefault)

	viewBar.JobIndicator = tview.NewTextView()
	viewBar.JobIndicator.SetDynamicColors(true)
	viewBar.JobIndicator.SetTextAlign(tview.AlignCenter)
	viewBar.JobIndicator.SetBackgroundColor(tcell.ColorDefault)

	go updateIndicators()

	viewBar.Flex = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(viewBar.Title, 0, 1, false).
		AddItem(viewBar.JobIndicator, 10, 0, false).
		AddItem(viewBar.ConnIndicator, 20, 0, false)

	return viewBar.Flex
}

// AddViews registers the views and their layouts.
func AddViews() {
	for _, view := range views {
		MainPage.AddPage(view.Name(), view.Layout(), true, false)
	}
}

// ShowViews shows a modal to select a view.
func ShowViews() {
	if page, _ := MainPage.GetFrontPage(); page == "show_view" {
		return
	}

	modal := NewModal("show_view", "Select page", false, false, len(views)+6, 60)

	modal.Table.SetSelectorWrap(false)
	modal.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := modal.Table.GetSelection()
			view := modal.Table.GetCell(row, 0).GetReference().(View)

			SetView(view)

			fallthrough

		case tcell.KeyEscape:
			modal.Exit()

		}

		return event
	})

	for row, view := range views {
		modal.Table.SetCell(row, 0, tview.NewTableCell(strings.Title(view.Name())).
			SetExpansion(1).
			SetReference(view).
			SetAlign(tview.AlignCenter),
		)
	}

	modal.Table.Select(0, 0)

	modal.Show()
}

// SetView brings the provided view into focus.
func SetView(view View, noexit ...struct{}) {
	if len(rclone.GetSessions()) == 0 {
		LoginScreen()
		return
	}

	if userInfo := cmd.GetConfigProperty("userInfo"); userInfo != "" {
		SetViewHostname(userInfo)
		cmd.AddConfigProperty("userInfo", "")
	}

	if noexit == nil && !currentView.Exit(view.Name()) {
		return
	}

	if !view.Init() {
		return
	}

	currentView = view

	MainPage.SwitchToPage(view.Name())
	SetViewTitle(view.Name())

	StopLoading()
}

// SetViewTitle sets the title for the current view.
func SetViewTitle(title string, appendToCurrent ...struct{}) {
	var currentTitle string

	title = strings.Title(title)
	if appendToCurrent != nil {
		currentTitle = viewBar.Title.GetText(true) + " " + title
	} else {
		currentTitle = "Rclone " + title
	}

	viewBar.Title.SetText("[::bu]" + currentTitle + "")
}

// SetViewHostname sets the user and host information within the title bar.
func SetViewHostname(userInfo string) {
	viewBar.ConnIndicator.SetText("[::b]" + userInfo)
	viewBar.Flex.ResizeItem(viewBar.ConnIndicator, len(userInfo)+2, 0)
}

// InitViewByName searches for a view by name and sets it.
func InitViewByName(name string) {
	for _, view := range views {
		if view.Name() == name {
			SetView(view, struct{}{})
			return
		}
	}
}

// updateIndicators updates the connectivity/job count indicators.
func updateIndicators() {
	for {
		select {
		case info := <-jobIndicator:
			App.QueueUpdateDraw(func() {
				var text string
				var color tcell.Color

				jobCount := info.JobCount
				if jobCount < 0 {
					return
				}
				if jobCount == 0 {
					text = ""
					color = tcell.ColorDefault
				} else {
					text = strconv.FormatInt(jobCount, 10)
					color = tcell.ColorYellow
				}

				viewBar.JobIndicator.SetText("[::b]" + text)
				viewBar.JobIndicator.SetBackgroundColor(color)
				viewBar.JobIndicator.SetTextColor(tcell.Color16)
			})

		case connected := <-rclone.PollConnection(false):
			App.QueueUpdateDraw(func() {
				var color tcell.Color

				if connected {
					color = tcell.ColorGreen
				} else {
					color = tcell.ColorRed
				}

				viewBar.ConnIndicator.SetBackgroundColor(color)
			})
		}
	}
}
