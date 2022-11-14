package ui

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/darkhz/tview"
	"github.com/darkhz/tvxwidgets"
	"github.com/gdamore/tcell/v2"
)

// Status stores the layout for a status bar.
type Status struct {
	Pages *tview.Pages

	Input   *tview.InputField
	Message *tview.TextView

	LoadText    *tview.TextView
	LoadSpinner *tvxwidgets.Spinner
	LoadStatus  bool

	msgchan     chan message
	loadchan    chan message
	messageLock sync.RWMutex
}

// message stores the status message.
type message struct {
	text    string
	persist bool
}

var (
	status Status

	sctx    context.Context
	scancel context.CancelFunc
)

// statusBar sets up the statusbar.
func statusBar() *tview.Pages {
	status.Pages = tview.NewPages()
	status.Pages.SetBackgroundColor(tcell.ColorDefault)

	status.Input = tview.NewInputField()
	status.Input.SetLabelColor(tcell.ColorWhite)
	status.Input.SetBackgroundColor(tcell.ColorDefault)
	status.Input.SetFieldBackgroundColor(tcell.ColorDefault)

	status.Message = tview.NewTextView()
	status.Message.SetDynamicColors(true)
	status.Message.SetBackgroundColor(tcell.ColorDefault)

	status.LoadSpinner = tvxwidgets.NewSpinner()
	status.LoadSpinner.SetStyle(tvxwidgets.SpinnerDotsCircling)
	status.LoadSpinner.SetBackgroundColor(tcell.ColorDefault)

	status.LoadText = tview.NewTextView()
	status.LoadText.SetDynamicColors(true)
	status.LoadText.SetBackgroundColor(tcell.ColorDefault)

	loadingFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(status.LoadSpinner, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(status.LoadText, 0, 1, false)

	status.Pages.AddPage("input", status.Input, true, true)
	status.Pages.AddPage("loading", loadingFlex, true, false)
	status.Pages.AddPage("messages", status.Message, true, false)

	status.Pages.SwitchToPage("messages")

	status.msgchan = make(chan message, 10)
	status.loadchan = make(chan message, 10)
	sctx, scancel = context.WithCancel(context.Background())

	go startStatus()
	go startLoad()

	return status.Pages
}

// StopStatus stops the message event loop.
func stopStatus() {
	scancel()
	close(status.msgchan)
}

// SetInput sets the inputfield label and returns the input text.
func SetInput(label string, multichar ...struct{}) string {
	entered := make(chan bool, 1)

	go func(ch chan bool) {
		send := func(state bool) {
			ch <- state

			_, item := MainPage.GetFrontPage()
			App.SetFocus(item)

			CloseStatusInput()
		}

		App.QueueUpdateDraw(func() {
			input := OpenStatusInput(label)
			input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					send(event.Key() == tcell.KeyEnter)
				}

				return event
			})
			if multichar != nil {
				input.SetAcceptanceFunc(nil)
			} else {
				input.SetAcceptanceFunc(tview.InputFieldMaxLength(1))
			}

			App.SetFocus(input)
		})
	}(entered)

	hasEntered := <-entered
	if !hasEntered {
		return ""
	}

	return status.Input.GetText()
}

// InfoMessage sends an info message to the status bar.
func InfoMessage(text string, persist bool) {
	if text != "" {
		text = "[white::b][I[] " + text
	}

	select {
	case status.msgchan <- message{text, persist}:
		return

	default:
	}
}

// ErrorMessage sends an error message to the status bar.
func ErrorMessage(page string, err error, stopLoading ...struct{}) {
	if stopLoading != nil {
		StopLoading()
	}

	if errors.Is(err, context.Canceled) {
		err = fmt.Errorf("Operation cancelled")
	}

	select {
	case status.msgchan <- message{
		"[red::b][E[] " + page + ": " + err.Error(), false,
	}:
		return

	default:
	}
}

// StartLoading starts the loading spinner.
func StartLoading(text string) {
	select {
	case status.loadchan <- message{text, true}:

	default:
	}
}

// StopLoading stops the loading spinner.
func StopLoading(text ...string) {
	var infoText string

	if text != nil {
		infoText = text[0]
	}

	select {
	case status.loadchan <- message{infoText, false}:

	default:
	}
}

// OpenStatusInput opens the input field within the status bar.
func OpenStatusInput(label string) *tview.InputField {
	status.Input.SetText("")
	status.Input.SetLabel("[::b]" + label + " ")

	status.Pages.SwitchToPage("input")

	return status.Input
}

// CloseStatusInput closes the input field within the status bar.
func CloseStatusInput() {
	var isLoading bool

	status.Input.SetChangedFunc(nil)
	status.Input.SetInputCapture(nil)
	status.Input.SetAcceptanceFunc(nil)

	status.messageLock.Lock()
	isLoading = status.LoadStatus
	status.messageLock.Unlock()

	if isLoading {
		status.Pages.SwitchToPage("loading")
		return
	}

	status.Pages.SwitchToPage("messages")
}

// ConfirmInput returns whether the user has pressed 'y' to confirm.
func ConfirmInput(label string) bool {
	reply := SetInput(label)

	return reply == "y"
}

// startLoad starts the spinner event loop.
func startLoad() {
	var spin bool

	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case load := <-status.loadchan:
			spin = load.persist

			status.messageLock.Lock()
			status.LoadStatus = load.persist
			status.messageLock.Unlock()

			App.QueueUpdateDraw(func() {
				if load.persist {
					switchStatusPage("loading")
					status.LoadText.SetText("[yellow::b]" + load.text)
				} else {
					status.LoadSpinner.Reset()
					switchStatusPage("messages")

					if load.text != "" {
						InfoMessage(load.text, false)
					}
				}
			})

		case <-t.C:
			if !spin {
				continue
			}

			App.QueueUpdateDraw(func() {
				status.LoadSpinner.Pulse()
			})
		}
	}
}

// startStatus starts the message event loop.
func startStatus() {
	var text string
	var cleared, loadPage bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-sctx.Done():
			return

		case msg, ok := <-status.msgchan:
			if !ok {
				return
			}

			t.Reset(2 * time.Second)

			cleared = false

			if msg.persist {
				text = msg.text
			}

			if !msg.persist && text != "" {
				text = ""
			}

			App.QueueUpdateDraw(func() {
				if page, _ := status.Pages.GetFrontPage(); page == "loading" {
					loadPage = true
					switchStatusPage("messages")
				}

				status.Message.SetText(msg.text)
			})

		case <-t.C:
			App.QueueUpdateDraw(func() {
				var isLoading bool

				if loadPage {
					loadPage = false

					status.messageLock.RLock()
					isLoading = status.LoadStatus
					status.messageLock.RUnlock()

					if isLoading {
						switchStatusPage("loading")
					}
				}

				if !cleared {
					cleared = true
					status.Message.SetText(text)
				}
			})
		}
	}
}

// switchStatusPage switches the status page only if the status bar
// input field is not in focus.
func switchStatusPage(page string) {
	if p, _ := status.Pages.GetFrontPage(); p == "input" {
		return
	}

	status.Pages.SwitchToPage(page)
}
