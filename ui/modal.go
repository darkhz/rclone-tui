package ui

import (
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Modal stores a layout to display a floating modal.
type Modal struct {
	Name string

	Flex     *tview.Flex
	Table    *tview.Table
	TextView *tview.TextView
	Input    *tview.InputField

	y *tview.Flex
	x *tview.Flex

	height, width int

	open bool
}

var currentModal *Modal

// NewModal returns a modal with an optional inputfield or textview.
func NewModal(name, title string, withInput, withTextView bool, height, width int) *Modal {
	var modalTable *tview.Table
	var modalTextView *tview.TextView

	modalTitle := tview.NewTextView()
	modalTitle.SetDynamicColors(true)
	modalTitle.SetText("[::bu]" + title)
	modalTitle.SetTextAlign(tview.AlignCenter)
	modalTitle.SetBackgroundColor(tcell.ColorBlack)

	if withTextView {
		modalTextView = tview.NewTextView()
		modalTextView.SetWrap(true)
		modalTextView.SetRegions(true)
		modalTextView.SetWordWrap(true)
		modalTextView.SetDynamicColors(true)
		modalTextView.SetBackgroundColor(tcell.ColorBlack)
	} else {
		modalTable = tview.NewTable()
		modalTable.SetSelectorWrap(true)
		modalTable.SetSelectable(true, false)
		modalTable.SetBackgroundColor(tcell.ColorBlack)
	}

	modalInput := tview.NewInputField()
	modalInput.SetBackgroundColor(tcell.ColorBlack)

	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetDirection(tview.FlexRow)

	box := tview.NewBox()
	box.SetBackgroundColor(tcell.ColorBlack)

	flex.AddItem(modalTitle, 1, 0, false)
	flex.AddItem(box, 1, 0, false)
	if withInput {
		flex.AddItem(modalInput, 1, 0, true)
	}
	flex.AddItem(box, 1, 0, false)
	if withTextView {
		flex.AddItem(modalTextView, 0, 1, !withInput)
	} else {
		flex.AddItem(modalTable, 0, 1, !withInput)
	}
	flex.SetBackgroundColor(tcell.ColorBlack)

	return &Modal{
		Name:     name,
		Flex:     flex,
		Input:    modalInput,
		Table:    modalTable,
		TextView: modalTextView,

		height: height,
		width:  width,
	}
}

// NewCustomModal returns a modal which includes the given item.
func NewCustomModal(name string, item tview.Primitive, height, width int) *Modal {
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(item, 0, 1, true)
	flex.SetBorder(true)
	flex.SetBackgroundColor(tcell.ColorDefault)

	return &Modal{
		Name: name,
		Flex: flex,

		height: height,
		width:  width,
	}
}

// Show shows the modal.
func (m *Modal) Show() {
	if currentModal != nil {
		return
	}

	prevPage, _ := MainPage.GetFrontPage()

	m.open = true

	m.y = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 0, false).
		AddItem(m.Flex, m.height, 0, true)

	m.x = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 0, false).
		AddItem(m.y, m.width, 0, true)

	MainPage.AddAndSwitchToPage(m.Name, m.x, true).ShowPage(prevPage)
	App.SetFocus(m.Flex)

	currentModal = m
	resizeModal()
}

// Exit exits the modal.
func (m *Modal) Exit() {
	MainPage.RemovePage(m.Name)

	currentModal = nil
}

// resizeModal resizes the modal according to the current screen dimensions.
func resizeModal() {
	if currentModal == nil {
		return
	}

	if !currentModal.open {
		return
	}

	_, _, pageWidth, pageHeight := MainPage.GetInnerRect()

	height := currentModal.height
	if height >= pageHeight {
		height = pageHeight
	}

	width := currentModal.width
	if width >= pageWidth {
		width = pageWidth
	}

	x := (pageWidth - currentModal.width) / 2
	y := (pageHeight - currentModal.height) / 2

	currentModal.y.ResizeItem(currentModal.Flex, height, 0)
	currentModal.y.ResizeItem(nil, y, 0)

	currentModal.x.ResizeItem(currentModal.y, width, 0)
	currentModal.x.ResizeItem(nil, x, 0)

	go App.Draw()
}
