package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// FormUI stores the layout for a form page.
type FormUI struct {
	ManagerTable   *tview.Table
	ManagerPages   *tview.Pages
	ManagerButtons []Button

	WizardPages   *tview.Pages
	WizardButtons []Button
	WizardData    map[string]interface{}
	WizardForms   map[string]*tview.Form
	WizardHelp    *tview.TextView

	Flex *tview.Flex

	dataLock    sync.RWMutex
	wizardLock  *semaphore.Weighted
	managerLock *semaphore.Weighted
}

// FormWidget stores the layout for a form item.
type FormWidget struct {
	tview.Primitive

	Item tview.FormItem

	Label  string
	Marker *tview.TextView

	listOptions *tview.TextView
}

// NewFormUI returns a form page. A form page typically consists of two sub-pages,
// the former to display data and the latter to configure data, like a wizard.
func NewFormUI(pages ...string) *FormUI {
	var formUI FormUI

	formUI.ManagerTable = tview.NewTable()
	formUI.ManagerTable.SetFixed(1, 1)
	formUI.ManagerTable.SetExpandSpace(true)
	formUI.ManagerTable.SetSelectable(true, false)
	formUI.ManagerTable.SetBackgroundColor(tcell.ColorDefault)

	formUI.WizardPages = tview.NewPages()
	formUI.WizardPages.SetBackgroundColor(tcell.ColorDefault)
	formUI.WizardPages.SetChangedFunc(func() {
		UpdateButtonView(formUI.WizardButtons)
	})

	formUI.WizardData = make(map[string]interface{})
	formUI.WizardForms = make(map[string]*tview.Form)

	for _, page := range pages {
		form := NewForm()

		formUI.WizardForms[page] = form
		formUI.WizardPages.AddPage(page, form, true, false)
	}

	formUI.WizardHelp = tview.NewTextView()
	formUI.WizardHelp.SetBorder(true)
	formUI.WizardHelp.SetDynamicColors(true)
	formUI.WizardHelp.SetBackgroundColor(tcell.ColorDefault)
	formUI.WizardHelp.SetFocusFunc(func() {
		formUI.WizardHelp.SetBorderColor(tcell.ColorBlue)
	})
	formUI.WizardHelp.SetBlurFunc(func() {
		formUI.WizardHelp.SetBorderColor(tcell.ColorWhite)
	})
	formUI.WizardHelp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlH:
			_, item := formUI.WizardPages.GetFrontPage()
			App.SetFocus(item)
		}

		return event
	})

	configWizardFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(TabView(), 1, 0, false).
		AddItem(formUI.WizardPages, 0, 1, true).
		AddItem(formUI.WizardHelp, 6, 0, false)

	formUI.ManagerPages = tview.NewPages()
	formUI.ManagerPages.AddPage("manager", formUI.ManagerTable, true, false)
	formUI.ManagerPages.AddPage("wizard", configWizardFlex, true, false)
	formUI.ManagerPages.SetBackgroundColor(tcell.ColorDefault)
	formUI.ManagerPages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		buttonEventHandler(event)

		if page, _ := formUI.ManagerPages.GetFrontPage(); page == "wizard" {
			tabEventHandler(event)
		}

		return event
	})

	formUI.wizardLock = semaphore.NewWeighted(1)
	formUI.managerLock = semaphore.NewWeighted(1)

	formUI.Flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(formUI.ManagerPages, 0, 1, true).
		AddItem(ButtonView(), 1, 0, false)
	formUI.Flex.SetBackgroundColor(tcell.ColorDefault)

	return &formUI
}

// NewForm returns a form. A form is a display area for various form items.
func NewForm() *tview.Form {
	form := tview.NewForm()
	form.SetBackgroundColor(tcell.ColorDefault)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers() != tcell.ModCtrl {
			goto Event
		}

		switch event.Key() {
		case tcell.KeyUp:
			form.InputHandler()(
				tcell.NewEventKey(tcell.KeyBacktab, ' ', tcell.ModNone), nil,
			)

		case tcell.KeyDown:
			form.InputHandler()(
				tcell.NewEventKey(tcell.KeyTab, ' ', tcell.ModNone), nil,
			)
		}

	Event:
		return event
	})

	return form
}

// NewFormWidget returns a form item, which can be added to a form.
func NewFormWidget(label string, primitive tview.FormItem, passwordButton ...*tview.Button) *FormWidget {
	labelView := tview.NewTextView()
	labelView.SetDynamicColors(true)
	labelView.SetText("[white::b]" + label + ":")
	labelView.SetBackgroundColor(tcell.ColorDefault)

	marker := tview.NewTextView()
	marker.SetDynamicColors(true)
	marker.SetBackgroundColor(tcell.ColorDefault)

	flex := tview.NewFlex()
	flex.AddItem(labelView, 0, 1, false)
	flex.AddItem(primitive, 0, 2, true)
	flex.AddItem(nil, 1, 0, false)
	flex.SetBackgroundColor(tcell.ColorDefault)

	if passwordButton != nil {
		flex.AddItem(passwordButton[0], 10, 0, false)
		flex.AddItem(nil, 1, 0, false)
	} else {
		flex.AddItem(nil, 10, 0, false)
		flex.AddItem(nil, 1, 0, false)
	}
	flex.AddItem(marker, 1, 0, false)

	return &FormWidget{
		Primitive: flex,
		Label:     label,
		Marker:    marker,
		Item:      primitive,
	}
}

// GetFormCheckBox returns a checkbox.
func GetFormCheckBox(
	label string,
	setData func(name string, data interface{}), doFunc func(label string),
	value ...string,
) *FormWidget {
	var f *FormWidget

	checkBox := tview.NewCheckbox()
	checkBox.SetBackgroundColor(tcell.ColorDefault)

	if value != nil {
		checkBox.SetChecked(value[0] == "true")
	}

	checkBox.SetChangedFunc(func(checked bool) {
		f.DisableMarker()
		setData(f.GetLabel(), checked)
	})
	checkBox.SetFocusFunc(func() {
		doFunc(f.GetLabel())
	})

	f = NewFormWidget(label, checkBox)

	return f
}

// GetFormInputField returns an input field.
func GetFormInputField(
	label string, editable, password bool,
	setData func(name string, data interface{}), doFunc func(label string),
	value ...string,
) *FormWidget {
	var f *FormWidget
	var b []*tview.Button
	var button *tview.Button

	inputField := tview.NewInputField()
	inputField.SetEnableFocus(true)
	inputField.SetBackgroundColor(tcell.ColorDefault)

	if value != nil {
		inputField.SetText(value[0])
	}
	if password {
		inputField.SetMaskCharacter('*')

		button = tview.NewButton("[::bu]Show")
		button.SetBackgroundColor(tcell.ColorDefault)
		button.SetSelectedFunc(func() {
			label := button.GetLabel()
			if strings.Index(label, "Show") != -1 {
				button.SetLabel("[::bu]Hide")
				inputField.SetMaskCharacter(0)
			} else {
				button.SetLabel("[::bu]Show")
				inputField.SetMaskCharacter('*')
			}
		})
	}

	inputField.SetChangedFunc(func(text string) {
		if !editable {
			return
		}

		setData(f.GetLabel(), text)
	})
	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyEnter:
			f.DisableMarker()
			return event

		case tcell.KeyCtrlP:
			if button != nil {
				button.InputHandler()(
					tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone), nil,
				)
			}

		default:
			if editable {
				goto Event
			}

			return nil
		}

	Event:
		return event
	})
	inputField.SetFocusFunc(func() {
		doFunc(f.GetLabel())
	})

	if button != nil {
		b = append(b, button)
	}

	f = NewFormWidget(label, inputField, b...)

	return f
}

// GetFormList returns a queryable list with options.
func GetFormList(
	label string,
	optionData map[string]string,
	editable, exclusive bool,
	setData func(name string, data interface{}),
	doFunc func(label string),
	changedFunc func(option string),
	value ...string,
) *FormWidget {
	var f *FormWidget

	input := tview.NewInputField()
	input.SetEnableFocus(true)
	input.SetBackgroundColor(tcell.ColorDefault)

	modal := GetList(
		"list_options", "Select "+label, label, optionData,
		func(text string) {
			input.SetText(text)
		}, changedFunc,
	)

	if value != nil {
		input.SetText(value[0])
	}

	input.SetChangedFunc(func(text string) {
		if !editable {
			return
		}

		setData(f.GetLabel(), text)
	})
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if !editable && event.Key() != tcell.KeyTab {
			return nil
		}

		switch event.Key() {
		case tcell.KeyDown, tcell.KeyUp:
			if event.Modifiers() == tcell.ModCtrl {
				goto Event
			}

			fallthrough

		case tcell.KeyCtrlO:
			modal.Show()

		case tcell.KeyRune:
			if exclusive {
				modal.Show()
				return nil
			}

		case tcell.KeyEnter:
			if input.GetText() == "" {
				modal.Show()
				f.DisableMarker()

				return nil
			}

		case tcell.KeyTab:
			f.DisableMarker()
		}

	Event:
		return event
	})
	input.SetFocusFunc(func() {
		doFunc(f.GetLabel())
	})

	button := tview.NewButton("[::bu]Options")
	button.SetBackgroundColor(tcell.ColorDefault)
	button.SetSelectedFunc(func() {
		if editable {
			modal.Show()
		}
	})

	f = NewFormWidget(label, input, button)

	return f
}

// GetList returns a list.
func GetList(
	name, title, label string,
	optionData map[string]string,
	exitFunc func(text string),
	changedFunc func(option string),
) *Modal {
	var options, highlight string

	modal := NewModal(
		name, title, true, true, len(optionData)+11, 60,
	)

	listInput, list := modal.Input, modal.TextView

	updateList := func(text string) {
		list.Clear()

		options, highlight = "", ""

		sortedKeys := []string{}
		for key := range optionData {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			if strings.Index(
				key, text,
			) == -1 &&
				strings.Index(
					strings.ToLower(optionData[key]),
					strings.ToLower(text),
				) == -1 {
				continue
			}

			if highlight == "" {
				highlight = key
			}

			optFormat := "[\"%s\"][::bu]%s[-:-:-][\"\"]\n%s\n"
			if optionData[key] != "" {
				optFormat += "\n"
			}

			options += fmt.Sprintf(
				optFormat,
				key, key, optionData[key],
			)
		}

		list.SetText(options)

		list.Highlight(highlight)
		list.ScrollToHighlight()
	}

	updateList("")

	listInput.SetChangedFunc(func(text string) {
		updateList(text)
	})
	listInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown, tcell.KeyUp:
			SwitchTabView(event.Key() == tcell.KeyUp, list)

		case tcell.KeyEnter:
			highlight := list.GetHighlights()
			if highlight != nil {
				exitFunc(highlight[0])
			}

			if changedFunc != nil {
				changedFunc(strings.ToLower(label))
			}

			fallthrough

		case tcell.KeyEscape:
			listInput.SetText("")
			modal.Exit()
		}

		return event
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		listInput.InputHandler()(event, nil)

		return event
	})

	return modal
}

// SetRequired marks the form item as required to be filled.
func (f *FormWidget) SetRequired() *FormWidget {
	f.Marker.SetText("[orange::bu]*[-:-:-]")

	return f
}

// EnableMarker enables the marker for the form item.
func (f *FormWidget) EnableMarker() *FormWidget {
	f.Marker.SetBackgroundColor(tcell.ColorRed)

	return f
}

// DisableMarker disables the marker for the form item.
func (f *FormWidget) DisableMarker() *FormWidget {
	f.Marker.SetBackgroundColor(tcell.ColorDefault)

	return f
}

// GetLabel returns the label for the form item.
func (f *FormWidget) GetLabel() string {
	return f.Label
}

// SetFormAttributes sets the form attributes for the form item.
func (f *FormWidget) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	formItem := f.GetFormItem()
	formItem.SetFormAttributes(labelWidth, labelColor, bgColor, fieldTextColor, fieldBgColor)

	return f
}

// GetFieldWidth returns the field width of the form item.
func (f *FormWidget) GetFieldWidth() int {
	_, _, w, _ := f.GetFormItem().GetRect()

	return w
}

func (f *FormWidget) GetFieldHeight() int {
	return 1
}

// SetFinishedFunc sets the handler for when the form item is not focused.
func (f *FormWidget) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	f.GetFormItem().SetFinishedFunc(handler)

	return f
}

// GetFormItem returns the form item.
func (f *FormWidget) GetFormItem() tview.FormItem {
	return f.Item
}
