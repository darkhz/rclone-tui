package ui

import (
	"strings"

	"github.com/darkhz/rclone-tui/cmd"
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	App *tview.Application

	MainPage *tview.Pages
	UIPages  *tview.Pages

	width, height int
	appSuspend    bool
)

// SetupUI sets up the user interface.
func SetupUI() {
	App = tview.NewApplication()
	UIPages = tview.NewPages()
	MainPage = tview.NewPages()

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ViewTitle(), 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(MainPage, 0, 1, true)
	flex.SetBackgroundColor(tcell.ColorDefault)

	UIPages.AddPage("main", flex, true, true)
	UIPages.SetBackgroundColor(tcell.ColorDefault)

	uiLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(UIPages, 0, 1, true).
		AddItem(statusBar(), 1, 0, false)
	uiLayout.SetBackgroundColor(tcell.ColorDefault)

	AddViews()
	if page := cmd.GetConfigProperty("page"); page != "" {
		InitViewByName(strings.Title(page))
	} else {
		InitViewByName("Dashboard")
	}

	go JobMonitor()

	App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			MainPage.InputHandler()(event, nil)
			return nil

		case tcell.KeyCtrlJ:
			openJobManager()

		case tcell.KeyCtrlN:
			ShowViews()

		case tcell.KeyCtrlX:
			if isOpen() {
				goto Event
			}

			rclone.CancelClientContext()

			if currentView != nil {
				page := currentView.Focused()
				if job, err := rclone.GetLatestJob("UI:" + page); err == nil {
					job.Cancel()
				}
			}

		case tcell.KeyCtrlZ:
			appSuspend = true

		case tcell.KeyCtrlH:
			ShowHelp()

		case tcell.KeyCtrlQ:
			go StopUI()
		}

	Event:
		return event
	})

	App.SetBeforeDrawFunc(func(t tcell.Screen) bool {
		suspendUI(t)

		return false
	})

	App.SetAfterDrawFunc(func(screen tcell.Screen) {
		w, h := screen.Size()

		if w == width && h == height {
			return
		}

		width, height = w, h

		resizeModal()
	})

	if err := App.SetRoot(uiLayout, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}

// StopUI asks for confirmation before stopping the user interface.
func StopUI() {
	if !ConfirmInput("Quit (y/n)?") {
		return
	}

	App.QueueUpdateDraw(func() {
		App.Stop()
	})
}

// suspendUI suspends the application.
func suspendUI(t tcell.Screen) {
	if !appSuspend {
		return
	}

	SuspendApp(t)

	appSuspend = false
}
