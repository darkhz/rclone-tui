package ui

import (
	"fmt"
	"sort"
	"strings"

	rcfns "github.com/darkhz/rclone-tui/rclone/operations"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/iancoleman/strcase"
)

// MountsUI stores a layout for the mounts page.
type MountsUI struct {
	formUI *FormUI
}

const (
	mountWizardTabs = `[aqua::b]["main"]Setup[""][-:-:-] [white::b]------[-:-:-] [aqua::b]["mountOpt"]Mount Options[""][-:-:-] [white::b]------[-:-:-] [aqua::b]["vfsOpt"]VFS Options[""][-:-:-]`
)

var mounts MountsUI

// Name returns the page's name.
func (m *MountsUI) Name() string {
	return "Mounts"
}

// Focused returns the currently focused view.
func (m *MountsUI) Focused() string {
	return m.Name()
}

// Init initializes the page.
func (m *MountsUI) Init() bool {
	m.formUI.ManagerPages.SwitchToPage("manager")
	go m.listMounts()

	return true
}

// Exit exits the page.
func (m *MountsUI) Exit(page string) bool {
	if pg, _ := m.formUI.ManagerPages.GetFrontPage(); pg == "wizard" {
		m.wizardExit(true, page)
		return false
	}

	m.formUI.WizardHelp.Clear()
	m.formUI.ManagerTable.Clear()

	m.clearWizardData()

	return true
}

// Layout returns this page's layout.
func (m *MountsUI) Layout() tview.Primitive {
	m.formUI = NewFormUI("main", "mountOpt", "vfsOpt")
	m.formUI.ManagerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			go m.listMounts()
		}

		switch event.Rune() {
		case 'n':
			m.managerNewMount()

		case 'u':
			m.managerUnmount()

		case 'U':
			m.managerUnmountAll()
		}

		return event
	})
	m.formUI.WizardPages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' && event.Modifiers() == tcell.ModAlt {
			App.SetFocus(m.formUI.WizardHelp)
			goto Event
		}

		switch event.Key() {
		case tcell.KeyCtrlS:
			m.wizardCreateMount()

		case tcell.KeyCtrlC:
			m.wizardCancel()
		}

	Event:
		return event
	})
	m.formUI.ManagerPages.SetChangedFunc(func() {
		page, _ := m.formUI.ManagerPages.GetFrontPage()

		switch page {
		case "manager":
			SetViewTitle("Mounts")

		case "wizard":
			SetViewTitle("Mount Wizard")
		}

		m.updateButtons(page)
	})

	m.formUI.ManagerButtons = []Button{
		{"New Mount", m.managerNewMount},
		{"Unmount", m.managerUnmount},
		{"Unmount All", m.managerUnmountAll},
	}

	m.formUI.WizardButtons = []Button{
		{"Next", m.wizardNext},
		{"Previous", m.wizardPrevious},
		{"Create Mount", m.wizardCreateMount},
		{"Cancel", m.wizardCancel},
	}

	strcase.ConfigureAcronym("Fs", "FS")

	return m.formUI.Flex
}

// listMounts lists the current mountpoints.
func (m *MountsUI) listMounts(filterSetting ...rcfns.MountPoint) {
	var err error
	var mountPoints []rcfns.MountPoint

	if filterSetting != nil {
		mountPoints = filterSetting
		goto LoadMountsTable
	}

	if !m.formUI.managerLock.TryAcquire(1) {
		return
	}
	defer m.formUI.managerLock.Release(1)

	StartLoading("Loading mountpoints")
	defer StopLoading()

	mountPoints, err = rcfns.GetMountPoints()

LoadMountsTable:
	sort.Slice(mountPoints, func(i, j int) bool {
		return mountPoints[i].Fs < mountPoints[j].Fs
	})

	App.QueueUpdateDraw(func() {
		m.formUI.ManagerTable.Clear()

		for col, header := range []string{
			"FS",
			"Mount Point",
			"Mounted On",
		} {
			m.formUI.ManagerTable.SetCell(0, col, tview.NewTableCell("[::bu]"+header).
				SetExpansion(1).
				SetSelectable(false).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(tcell.ColorPurple),
			)
		}

		if err != nil {
			ErrorMessage("Mounts", err)
			return
		}

		bgColor := tcell.ColorSlateGray
		row := m.formUI.ManagerTable.GetRowCount() - 1

		for _, point := range mountPoints {
			row++

			m.formUI.ManagerTable.SetCell(row, 0, tview.NewTableCell(tview.Escape(point.Fs)).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(bgColor),
			)
			m.formUI.ManagerTable.SetCell(row, 1, tview.NewTableCell(point.MountPoint).
				SetMaxWidth(10).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(bgColor),
			)
			m.formUI.ManagerTable.SetCell(row, 2, tview.NewTableCell(point.MountedOn.Format("Mon 01/02 15:04")).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(bgColor),
			)
		}

		m.updateButtons("manager")

		if filterSetting == nil {
			App.SetFocus(m.formUI.ManagerTable)
		}
	})
}

// managerNewMount displays the mount wizard.
func (m *MountsUI) managerNewMount() {
	go m.setupWizard()
}

// managerUnmount unmounts the selected mountpoint.
func (m *MountsUI) managerUnmount() {
	selectedRow, _ := m.formUI.ManagerTable.GetSelection()
	if selectedRow <= 0 {
		return
	}

	mountedOn := m.formUI.ManagerTable.GetCell(selectedRow, 1).Text

	go func(row int, mountpoint string) {
		if !m.formUI.wizardLock.TryAcquire(1) {
			return
		}
		defer m.formUI.wizardLock.Release(1)

		if !ConfirmInput("Unmount " + mountpoint + "? (y/n)") {
			return
		}

		StartLoading("Unmounting" + mountpoint)

		if err := rcfns.Unmount(mountpoint); err != nil {
			ErrorMessage("Mounts", err, struct{}{})
			return
		}

		StopLoading("Unmounted" + mountpoint)

		App.QueueUpdateDraw(func() {
			m.formUI.ManagerTable.RemoveRow(row)
			m.formUI.ManagerTable.Select(row, 0)

			m.updateButtons("manager")
		})
	}(selectedRow, mountedOn)
}

// managerUnmountAll unmounts all the mountpoints.
func (m *MountsUI) managerUnmountAll() {
	go func() {
		if !m.formUI.wizardLock.TryAcquire(1) {
			return
		}
		defer m.formUI.wizardLock.Release(1)

		if !ConfirmInput("Unmount all mountpoints? (y/n)") {
			return
		}

		StartLoading("Unmounting all mountpoints")

		if err := rcfns.UnmountAll(); err != nil {
			ErrorMessage("Mounts", err, struct{}{})
			return
		}

		StopLoading("Unmounted all mountpoints")

		App.QueueUpdateDraw(func() {
			for row := 1; row < m.formUI.ManagerTable.GetRowCount(); row++ {
				m.formUI.ManagerTable.RemoveRow(row)
			}

			m.updateButtons("manager")
		})
	}()
}

// wizardNext moves to the next page of the form.
func (m *MountsUI) wizardNext() {
	SwitchTabView(false)
	App.SetFocus(m.formUI.WizardPages)
}

// wizardPrevious moves to the previous page of the form.
func (m *MountsUI) wizardPrevious() {
	SwitchTabView(true)
	App.SetFocus(m.formUI.WizardPages)
}

// wizardCreateMount creates a mountpoint.
func (m *MountsUI) wizardCreateMount() {
	go func() {
		if !m.formUI.wizardLock.TryAcquire(1) {
			return
		}
		defer m.formUI.wizardLock.Release(1)

		var fs, mountPoint string

		if f, ok := m.formUI.WizardData["fs"].(string); ok && f != "" {
			fs = f
		} else {
			ErrorMessage("Mounts", fmt.Errorf("FS is not specified"))
			return
		}

		if m, ok := m.formUI.WizardData["mountPoint"].(string); ok && m != "" {
			mountPoint = m
		} else {
			ErrorMessage("Mounts", fmt.Errorf("Mount Point is not specified"))
			return
		}

		loadText := fs + " on " + mountPoint

		StartLoading("Mounting " + loadText)

		if err := rcfns.CreateMount(parseDataMap(m.formUI.WizardData)); err != nil {
			ErrorMessage("Mounts", err, struct{}{})
			return
		}

		StopLoading("Mounted " + loadText)

		m.wizardExit(false)
	}()
}

// wizardCancel asks for confirmation before exiting the wizard.
func (m *MountsUI) wizardCancel() {
	m.wizardExit(true)
}

// wizardExit exits the wizard.
func (m *MountsUI) wizardExit(confirm bool, page ...string) {
	go func() {
		if confirm && !ConfirmInput("Cancel configuration editing (y/n)?") {
			return
		}

		m.clearWizardData()

		if page != nil {
			App.QueueUpdateDraw(func() {
				InitViewByName(page[0])
			})

			return
		}

		App.QueueUpdateDraw(func() {
			m.formUI.ManagerPages.SwitchToPage("manager")
		})

		m.listMounts()
	}()
}

// setupWizard sets up the mounts wizard.
func (m *MountsUI) setupWizard() {
	StartLoading("Loading mountpoints")

	mountHelp, err := rcfns.GetMountOptions()
	if err != nil {
		ErrorMessage("Mounts", err, struct{}{})
		return
	}

	StopLoading()

	for _, opt := range mountHelp {
		var page string
		var formItem *FormWidget

		if strings.HasPrefix(opt.OptionType, "main") {
			page = "main"
		} else {
			page = opt.OptionType
		}

		name := strings.Title(strcase.ToDelimited(opt.Name, ' '))

		switch {
		case opt.ValueType == "bool":
			formItem = GetFormCheckBox(
				name,
				m.setWizardData, m.updateHelp,
				m.getWizardData(name),
			)

		case opt.Options != nil:
			optionData := map[string]string{}

			for _, o := range opt.Options {
				optionData[o] = ""
			}

			formItem = GetFormList(
				name, optionData, true, opt.Name == "FS",
				m.setWizardData, m.updateHelp, nil,
				m.getWizardData(name),
			)

		default:
			formItem = GetFormInputField(
				name, true, false,
				m.setWizardData, m.updateHelp,
				m.getWizardData(name),
			)
		}

		if strings.HasSuffix(opt.OptionType, "required") {
			formItem.SetRequired()
		}

		if formItem != nil {
			App.QueueUpdateDraw(func() {
				m.formUI.WizardForms[page].AddFormItem(formItem)
			})
		}
	}

	App.QueueUpdateDraw(func() {
		SetupTabs(
			mountWizardTabs, tview.AlignCenter,
			func(reverse bool) bool {
				return true
			},
			func(tab string) {
				m.formUI.WizardPages.SwitchToPage(tab)
			},
		)

		m.formUI.ManagerPages.SwitchToPage("wizard")
		m.formUI.WizardPages.SwitchToPage("main")

		App.SetFocus(m.formUI.WizardPages)
	})
}

// updateButtons updates the buttons according to the page/form displayed.
func (m *MountsUI) updateButtons(page string) {
	if pg, _ := m.formUI.ManagerPages.GetFrontPage(); pg != page {
		return
	}

	switch page {
	case "manager":
		UpdateButtonView(m.formUI.ManagerButtons, func(label string) bool {
			if m.formUI.ManagerTable.GetRowCount() == 0 {
				return false
			}

			if m.formUI.ManagerTable.GetRowCount() <= 1 && label != "New Mount" {
				return false
			}

			return true
		})

	case "wizard":
		UpdateButtonView(m.formUI.WizardButtons, func(label string) bool {
			page, _ := m.formUI.WizardPages.GetFrontPage()

			switch page {
			case "main":
				if label == "Previous" {
					return false
				}

			case "vfsOpt":
				if label == "Next" {
					return false
				}
			}

			return true
		})
	}
}

// updateHelp updates the help information for each item within the form.
func (m *MountsUI) updateHelp(field string) {
	mh := rcfns.GetMountHelp(strcase.ToCamel(field))
	name := strings.Title(strcase.ToDelimited(mh.Name, ' '))

	text := "[::bu]" + name + "[-:-:-]"
	if strings.HasSuffix(mh.OptionType, "required") {
		text += " (Required)"
	}

	text += "\n" + mh.Help

	m.formUI.WizardHelp.SetText(text)
	m.formUI.WizardHelp.ScrollToBeginning()
}

// getWizardData returns the stored form data.
func (m *MountsUI) getWizardData(key string) string {
	m.formUI.dataLock.RLock()
	defer m.formUI.dataLock.RUnlock()

	dataKey, dataMap := m.getDataMap(key)

	return modifyDataMap(dataMap, dataKey, nil, false)
}

// setWizardData stores a form item's name(key) and its value.
func (m *MountsUI) setWizardData(key string, value interface{}) {
	m.formUI.dataLock.Lock()
	defer m.formUI.dataLock.Unlock()

	dataKey, dataMap := m.getDataMap(key)

	modifyDataMap(dataMap, dataKey, value, true)
}

// clearWizardData clears the form data.
func (m *MountsUI) clearWizardData() {
	m.formUI.dataLock.Lock()
	defer m.formUI.dataLock.Unlock()

	m.formUI.WizardData = make(map[string]interface{})

	for _, form := range m.formUI.WizardForms {
		form.Clear(true)
	}
}

// getDataMap returns either the nested data for mount or vfs options, or
// it returns the wizard map itself for other(main) options.
func (m *MountsUI) getDataMap(key string) (string, map[string]interface{}) {
	dataMap := m.formUI.WizardData

	if page, _ := m.formUI.WizardPages.GetFrontPage(); page == "mountOpt" || page == "vfsOpt" {
		key = strcase.ToCamel(key)

		v, ok := m.formUI.WizardData[page]
		if !ok || ok && v == nil {
			dataMap[page] = make(map[string]interface{})
		}

		dataMap = dataMap[page].(map[string]interface{})
	} else if page == "main" {
		if strings.ToLower(key) == "fs" {
			key = "fs"
		} else {
			key = strcase.ToLowerCamel(key)
		}
	}

	return key, dataMap
}
