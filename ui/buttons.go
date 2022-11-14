package ui

import (
	"fmt"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Button stores a button's label and the function to be
// executed when it is selected.
type Button struct {
	label        string
	selectedFunc func()
}

// ButtonDisplay stores the layout for the currently configured buttons.
type ButtonDisplay struct {
	Layout *tview.TextView

	currentButtons []Button
	lastFocus      tview.Primitive

	arrangeFunc func(label string) bool
}

var buttonDisplay ButtonDisplay

// ButtonView returns a display area to show buttons.
func ButtonView() *tview.TextView {
	if buttonDisplay.Layout != nil {
		goto ShowButtonLayout
	}

	buttonDisplay.Layout = tview.NewTextView()
	buttonDisplay.Layout.SetRegions(true)
	buttonDisplay.Layout.SetDynamicColors(true)
	buttonDisplay.Layout.SetTextAlign(tview.AlignRight)
	buttonDisplay.Layout.SetBackgroundColor(tcell.ColorDefault)
	buttonDisplay.Layout.SetBlurFunc(func() {
		buttonDisplay.Layout.Highlight("")
	})
	buttonDisplay.Layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			if event.Modifiers() == tcell.ModCtrl {
				switchButtons(event.Key() == tcell.KeyLeft)
				goto Event
			}

		case tcell.KeyEnter:
			var selected string

			highlights := buttonDisplay.Layout.GetHighlights()
			if highlights != nil {
				selected = highlights[0]
			}

			for _, button := range buttonDisplay.currentButtons {
				if button.label == selected {
					button.selectedFunc()
					buttonDisplay.Layout.Highlight("")

					goto Event
				}
			}
		}

		if buttonDisplay.lastFocus != nil {
			App.SetFocus(buttonDisplay.lastFocus)
			buttonDisplay.Layout.Highlight("")
		}

	Event:
		return event
	})

ShowButtonLayout:
	return buttonDisplay.Layout
}

// UpdateButtonView updates the button display area. If addCondition is provided,
// each button label is checked against addCondition before displaying it.
func UpdateButtonView(buttons []Button, addCondition ...func(label string) bool) {
	var buttonText string

	if buttons == nil {
		return
	}

	buttonDisplay.currentButtons = buttons

	ButtonView().Clear()

	for _, button := range buttons {
		if addCondition != nil {
			buttonDisplay.arrangeFunc = addCondition[0]
		}

		if buttonDisplay.arrangeFunc != nil && !buttonDisplay.arrangeFunc(button.label) {
			continue
		}

		buttonText += fmt.Sprintf("[\"%s\"][blue::b][%s[][-:-:-][\"\"] ", button.label, button.label)
	}

	ButtonView().SetText(buttonText)
}

// switchButtons cycles between the button selection.
func switchButtons(reverse bool) {
	SwitchTabView(reverse, buttonDisplay.Layout)
}

// focusButtonView brings the button display area into focus.
func focusButtonView() {
	buttonDisplay.lastFocus = App.GetFocus()

	App.SetFocus(ButtonView())

	switchButtons(false)
}

// buttonEventHandler handles key events for the button display area.
func buttonEventHandler(event *tcell.EventKey) bool {
	if event.Modifiers() != tcell.ModCtrl {
		return false
	}

	switch event.Key() {
	case tcell.KeyLeft, tcell.KeyRight:
		focusButtonView()
	}

	return true
}
