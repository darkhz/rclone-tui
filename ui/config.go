package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// ConfigUI stores a layout for the configuration page.
type ConfigUI struct {
	formUI *FormUI

	checkedName    string
	formLoaded     bool
	WizardUpdating bool
	options        []rclone.ProviderOption
}

const (
	configWizardTabs        = `[aqua::b]["setup"]Setup[""][-:-:-] [white::b]------[-:-:-] [aqua::b]["basic"]Basic[""][-:-:-] `
	configWizardAdvancedTab = `[white::b]------[-:-:-] [aqua::b]["advanced"]Advanced[""][-:-:-]`
)

var configuration ConfigUI

// Name returns the page's name.
func (c *ConfigUI) Name() string {
	return "Configuration"
}

// Focused returns the currently focused view.
func (c *ConfigUI) Focused() string {
	return c.Name()
}

// Init initializes the page.
func (c *ConfigUI) Init() bool {
	c.formUI.ManagerPages.SwitchToPage("manager")
	go c.listConfigSettings()

	return true
}

// Exit exits the page.
func (c *ConfigUI) Exit(page string) bool {
	if page, _ := c.formUI.ManagerPages.GetFrontPage(); page == "wizard" {
		c.wizardCancel()

		return false
	}

	c.formUI.WizardHelp.Clear()
	c.formUI.ManagerTable.Clear()

	c.clearWizardData()

	return true
}

// Layout returns this page's layout.
func (c *ConfigUI) Layout() tview.Primitive {
	c.formUI = NewFormUI("setup", "basic", "advanced")
	c.formUI.ManagerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			go c.listConfigSettings()
		}

		switch event.Rune() {
		case 'n':
			c.managerCreateNew()

		case 'u':
			c.managerUpdate()

		case 'd':
			c.managerDelete()

		case '/':
			c.managerFilter()
		}

		return event
	})
	c.formUI.WizardPages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 's' && event.Modifiers() == tcell.ModAlt {
			go c.saveConfig(true)
			goto Event
		}

		switch event.Key() {
		case tcell.KeyCtrlF:
			c.wizardJump()

		case tcell.KeyCtrlS:
			go c.saveConfig(false)

		case tcell.KeyCtrlC:
			c.wizardCancel()

		case tcell.KeyCtrlH:
			App.SetFocus(c.formUI.WizardHelp)
		}

	Event:
		return event
	})
	c.formUI.ManagerPages.SetChangedFunc(func() {
		page, _ := c.formUI.ManagerPages.GetFrontPage()

		switch page {
		case "manager":
			SetViewTitle("Configuration")

		case "wizard":
			SetViewTitle("Configuration Wizard")
		}

		c.updateButtons(page)
	})

	c.formUI.ManagerButtons = []Button{
		{"Create New", c.managerCreateNew},
		{"Update", c.managerUpdate},
		{"Delete", c.managerDelete},
	}

	c.formUI.WizardButtons = []Button{
		{"Next", c.wizardNext},
		{"Previous", c.wizardPrevious},
		{"Filter", c.wizardJump},
		{"Save", c.wizardSave},
		{"Cancel", c.wizardCancel},
	}

	return c.formUI.Flex
}

// listConfigSettings lists the rclone configurations.
func (c *ConfigUI) listConfigSettings(filterSetting ...map[string]map[string]interface{}) {
	var err error
	var settingKeys []string
	var settings map[string]map[string]interface{}

	if filterSetting != nil {
		settings = filterSetting[0]
		goto LoadConfigTable
	}

	if !c.formUI.managerLock.TryAcquire(1) {
		return
	}
	defer c.formUI.managerLock.Release(1)

	StartLoading("Loading configuration settings...")
	defer StopLoading()

	settings, err = rclone.GetConfigSettings()

LoadConfigTable:
	for name := range settings {
		settingKeys = append(settingKeys, name)
	}
	sort.Strings(settingKeys)

	App.QueueUpdateDraw(func() {
		c.formUI.ManagerTable.Clear()

		for col, header := range []string{
			"Name",
			"Type",
		} {
			c.formUI.ManagerTable.SetCell(0, col, tview.NewTableCell("[::bu]"+header).
				SetExpansion(1).
				SetSelectable(false).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(tcell.ColorPurple),
			)
		}

		if err != nil {
			ErrorMessage("Configuration", err)
			return
		}

		bgColor := tcell.ColorSlateGray
		row := c.formUI.ManagerTable.GetRowCount() - 1

		for _, key := range settingKeys {
			var configDesc string
			setting := settings[key]

			configType := setting["type"].(string)
			configDesc = rclone.GetProviderDesc(configType)

			if configType == "s3" {
				descFields := strings.Fields(configDesc)
				for i, field := range descFields {
					if field == "including" {
						descFields = descFields[:i]
						break
					}
				}

				configDesc = strings.Join(descFields, " ")
			}

			row++

			c.formUI.ManagerTable.SetCell(row, 0, tview.NewTableCell(tview.Escape(key)).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(bgColor),
			)
			c.formUI.ManagerTable.SetCell(row, 1, tview.NewTableCell(configDesc).
				SetMaxWidth(10).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(bgColor),
			)
		}

		c.updateButtons("manager")

		if filterSetting == nil {
			App.SetFocus(c.formUI.ManagerTable)
		}
	})
}

// managerCreateNew shows the configuration wizard.
func (c *ConfigUI) managerCreateNew() {
	c.setWizardUpdating(false)

	go c.setupWizard(true)
}

// managerUpdate updates the selected configuration.
func (c *ConfigUI) managerUpdate() {
	row, _ := c.formUI.ManagerTable.GetSelection()
	if row <= 0 {
		return
	}

	name := c.formUI.ManagerTable.GetCell(row, 0).Text
	config := c.formUI.ManagerTable.GetCell(row, 1).Text

	c.setWizardData("name", name)
	c.setWizardData("configuration", config)

	c.setWizardUpdating(true)

	go c.setupWizard(false)
}

// managerFilter filters through the configuration and displays the result.
func (c *ConfigUI) managerFilter() {
	input := OpenStatusInput("Filter config:")
	input.SetChangedFunc(func(text string) {
		filterSetting := rclone.GetCurrentSettings()

		for name := range filterSetting {
			if strings.Index(name, text) == -1 {
				delete(filterSetting, name)
			}
		}

		go c.listConfigSettings(filterSetting)
	})
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			CloseStatusInput()
			App.SetFocus(c.formUI.ManagerTable)

			if c.formUI.ManagerTable.GetRowCount() <= 1 {
				go c.listConfigSettings(rclone.GetCurrentSettings())
			}
		}

		return event
	})

	App.SetFocus(input)
}

// managerDelete deletes the selected configuration.
func (c *ConfigUI) managerDelete() {
	currentRow, _ := c.formUI.ManagerTable.GetSelection()
	if currentRow <= 0 {
		return
	}

	configName := c.formUI.ManagerTable.GetCell(currentRow, 0).Text

	go func(row int, name string) {
		if !ConfirmInput("Delete configuration '" + name + "' (y/n)?") {
			return
		}

		if !c.formUI.managerLock.TryAcquire(1) {
			InfoMessage("Deletion in progress...", false)
			return
		}
		defer c.formUI.managerLock.Release(1)

		StartLoading("Deleting configuration '" + name + "'")

		err := rclone.DeleteConfig(name)
		if err != nil {
			ErrorMessage("Configuration", err, struct{}{})
		}

		StopLoading("Deleted configuration '" + name + "'")

		App.QueueUpdateDraw(func() {
			c.formUI.ManagerTable.RemoveRow(row)
			c.formUI.ManagerTable.Select(row, 0)

			c.updateButtons("manager")
		})
	}(currentRow, configName)
}

// wizardJump jumps to the selected option in the form.
func (c *ConfigUI) wizardJump() {
	var row int
	var filterOptions []rclone.ProviderOption

	if len(c.options) == 0 {
		return
	}

	modal := NewModal(
		"filter_options", "Jump to option", true, false, 20, 60,
	)

	filterInput, filterTable := modal.Input, modal.Table

	updateTable := func(row int, option rclone.ProviderOption) bool {
		if !MatchProvider(option.Provider, c.getWizardData("provider")) {
			return false
		}

		displayOption := "[aqua::b]" + option.Name
		if option.Advanced {
			displayOption += "[purple::b] (Advanced)"
		} else {
			displayOption += "[pink::b] (Basic)"
		}

		filterTable.SetCell(row, 0, tview.NewTableCell(displayOption).
			SetReference(option).
			SetAttributes(tcell.AttrBold).
			SetSelectedStyle(tcell.Style{}.Reverse(true)).
			SetClickedFunc(func() bool {
				filterInput.InputHandler()(
					tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone), nil,
				)

				return true
			}),
		)

		return true
	}

	selectForm := func() {
		var formType string

		row, _ := filterTable.GetSelection()
		option, ok := filterTable.GetCell(row, 0).
			GetReference().(rclone.ProviderOption)
		if !ok {
			return
		}

		if option.Advanced {
			formType = "advanced"
		} else {
			formType = "basic"
		}

		SelectTab(formType)

		form := c.formUI.WizardForms[formType]
		form.SetFocus(form.GetFormItemIndex(option.Name))
	}

	filterOptions = append(filterOptions, c.options...)
	sort.Slice(filterOptions, func(i, j int) bool {
		return !(!filterOptions[i].Advanced != !filterOptions[j].Advanced)
	})

	for _, option := range filterOptions {
		if updateTable(row, option) {
			row++
		}
	}

	filterInput.SetLabel("Filter: ")

	filterInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown, tcell.KeyUp:
			filterTable.InputHandler()(event, nil)

		case tcell.KeyEnter:
			selectForm()
			fallthrough

		case tcell.KeyEscape:
			modal.Exit()
		}

		return event
	})
	filterInput.SetChangedFunc(func(text string) {
		var row int

		filterTable.Clear()

		for _, option := range filterOptions {
			if strings.Index(option.Name, text) == -1 {
				continue
			}

			if updateTable(row, option) {
				row++
			}
		}

		filterTable.ScrollToBeginning()
		filterTable.Select(0, 0)
	})

	modal.Show()
}

// wizardNext goes to the next page of the form.
func (c *ConfigUI) wizardNext() {
	SwitchTabView(false)
	App.SetFocus(c.formUI.WizardPages)
}

// wizardPrevious goes to the previous page of the form.
func (c *ConfigUI) wizardPrevious() {
	SwitchTabView(true)
	App.SetFocus(c.formUI.WizardPages)
}

// wizardSave asks whether to automatically authorize the configuration remote,
// and saves the configuration.
func (c *ConfigUI) wizardSave() {
	go c.saveConfig(false, struct{}{})
}

// saveConfig saves the configuration.
func (c *ConfigUI) saveConfig(interactiveConfig bool, confirm ...struct{}) {
	if !c.formUI.wizardLock.TryAcquire(1) {
		InfoMessage("Saving in progress...", false)
		return
	}
	defer c.formUI.wizardLock.Release(1)

	if confirm != nil {
		interactiveConfig = ConfirmInput("Authorize interactively? (y/n)")
	}

	StartLoading("Saving configuration...")

	err := rclone.SaveConfig(
		parseDataMap(c.formUI.WizardData), !c.getWizardUpdating(), interactiveConfig,
	)
	if err != nil {
		ErrorMessage("Configuration", err, struct{}{})
		return
	}

	StopLoading("Configuration saved")

	c.wizardExit(false)
}

// wizardCancel asks for confirmation before exiting the wizard.
func (c *ConfigUI) wizardCancel() {
	c.wizardExit(true)
}

// wizardExit exits the configuration wizard.
func (c *ConfigUI) wizardExit(confirm bool) {
	go func() {
		if confirm && !ConfirmInput("Cancel configuration editing (y/n)?") {
			return
		}

		App.QueueUpdateDraw(func() {
			c.formUI.ManagerPages.SwitchToPage("manager")
		})

		c.clearWizardData()
		c.listConfigSettings()
	}()
}

// setupWizard sets up the configuration wizard.
func (c *ConfigUI) setupWizard(newConfig bool, providerUpdate ...struct{}) {
	var err error
	var provider rclone.Provider
	var config map[string]map[string]interface{}

	if !c.formUI.wizardLock.TryAcquire(1) {
		return
	}
	defer c.formUI.wizardLock.Release(1)

	StartLoading("Creating forms...")
	defer StopLoading()

	providers, err := rclone.GetConfigProviders()
	if err != nil {
		ErrorMessage("Configuration", err, struct{}{})
		return
	}

	if !newConfig {
		c.setFormLoaded(false)

		config, err = rclone.GetConfigSettings()
		if err != nil {
			ErrorMessage("Configuration", err, struct{}{})
			return
		}

		if config := c.getWizardData("configuration"); config != "" {
			provider, err = rclone.GetProviderByDesc(config)
			if err != nil {
				ErrorMessage("Configuration", err, struct{}{})
				return
			}

			c.setWizardData("type", provider.Prefix)
		} else {
			provider, err = rclone.GetProviderByType(c.getWizardData("type"))
			if err != nil {
				ErrorMessage("Configuration", err, struct{}{})
				return
			}
		}
	}

	App.QueueUpdateDraw(func() {
		c.formUI.ManagerPages.SwitchToPage("wizard")

		c.createSetupForm(providers)

		if providerUpdate == nil {
			SetupTabs(
				configWizardTabs, tview.AlignCenter, c.wizardFormHandler,
				func(tab string) {
					c.formUI.WizardPages.SwitchToPage(tab)
				},
			)
		}

		if !newConfig {
			err := c.createBasicAdvancedForm(config, provider, providerUpdate...)
			if err != nil {
				ErrorMessage("Configuration", err, struct{}{})
				return
			}

			SelectTab("basic")
			c.setFormLoaded(true)
		} else {
			SelectTab("setup")
		}
	})
}

// wizardFormHandler decides when to move to another page within
// the form. This is used in the tab display handler.
func (c *ConfigUI) wizardFormHandler(reverse bool) bool {
	var formError bool

	if reverse {
		return true
	}

	page, item := c.formUI.WizardPages.GetFrontPage()

	form, ok := item.(*tview.Form)
	if !ok {
		ErrorMessage(
			"Configuration",
			fmt.Errorf("Could not parse %s form", page),
		)

		return false
	}

	defer App.SetFocus(form)

	for i := 0; i < form.GetFormItemCount(); i++ {
		var options []rclone.ProviderOption

		formItem := form.GetFormItem(i).(*FormWidget)

		options = append(options, c.options...)
		if page == "setup" {
			options = append(options, []rclone.ProviderOption{
				{Name: "name", Required: true},
				{Name: "type", Required: true},
			}...)
		}
		if options == nil {
			ErrorMessage(
				"Configuration",
				fmt.Errorf("Could not get provider options"),
			)

			return false
		}

		for _, option := range options {
			if option.Name == strings.ToLower(formItem.GetLabel()) {
				if option.Required && c.checkWizardDataEmpty(option.Name) != nil {
					formError = true
					formItem.EnableMarker()
				}
			}
		}
	}

	if formError {
		ErrorMessage(
			"Configuration",
			fmt.Errorf("Fill the required fields"),
		)

		return false
	}

	if page == "setup" {
		go c.setupWizard(false)
		if !c.getFormLoaded() {
			return false
		}
	}

	return true
}

// createSetupForm creates the initial basic form.
func (c *ConfigUI) createSetupForm(providers rclone.ConfigProviders) {
	optionData := map[string]string{}

	for _, provider := range providers.Providers {
		optionData[provider.Prefix] = provider.Description
	}

	setupForm := c.formUI.WizardForms["setup"].Clear(false)

	setupForm.AddFormItem(
		GetFormInputField(
			"Name", !c.getWizardUpdating(), false,
			c.setWizardData, c.updateHelp,
			c.getWizardData("name"),
		).SetRequired(),
	)

	setupForm.AddFormItem(
		GetFormList(
			"Type", optionData, !c.getWizardUpdating(), true,
			c.setWizardData, c.updateHelp, nil,
			c.getWizardData("type"),
		).SetRequired(),
	)

	App.SetFocus(setupForm)
}

// createBasicAdvancedForm creates basic and advanced forms based on the options
// provided in the setup(basic) form.
//
//gocyclo:ignore
func (c *ConfigUI) createBasicAdvancedForm(
	config map[string]map[string]interface{},
	provider rclone.Provider,
	providerUpdate ...struct{},
) error {
	if err := c.checkWizardDataEmpty("name", "type"); err != nil {
		return err
	}

	name := c.getWizardData("name")

	setting := config[name]
	if providerUpdate == nil {
		if c.getWizardUpdating() {
			for settingKey, settingValue := range setting {
				c.setWizardData(settingKey, settingValue)
			}
		} else {
			if setting != nil && name != c.checkedName {
				return fmt.Errorf("'" + name + "' already exists")
			} else {
				c.checkedName = name
			}
		}
	}

	basicForm := c.formUI.WizardForms["basic"].Clear(false)
	advancedForm := c.formUI.WizardForms["advanced"]
	if providerUpdate == nil {
		advancedForm.Clear(false)
	}

	c.options = provider.Options

	for _, option := range provider.Options {
		var formItem *FormWidget

		if providerUpdate != nil && option.Advanced {
			continue
		}

		if !MatchProvider(option.Provider, c.getWizardData("provider")) {
			continue
		}

		switch {
		case option.Type == "bool":
			formItem = GetFormCheckBox(
				option.Name,
				c.setWizardData, c.updateHelp,
				c.getWizardData(option.Name),
			)

		case option.Examples != nil:
			optionData := map[string]string{}

			for _, example := range option.Examples {
				if !MatchProvider(example.Provider, c.getWizardData("provider")) {
					continue
				}

				if example.Value == "" {
					continue
				}

				optionData[example.Value] = example.Help
			}

			formItem = GetFormList(
				option.Name, optionData, true, option.Exclusive,
				c.setWizardData, c.updateHelp, func(label string) {
					if label == "provider" {
						go c.setupWizard(false, struct{}{})
					}
				}, c.getWizardData(option.Name),
			)

		default:
			formItem = GetFormInputField(
				option.Name, true, option.IsPassword,
				c.setWizardData, c.updateHelp,
				c.getWizardData(option.Name),
			)
		}

		if option.Required {
			formItem.SetRequired()
		}

		if formItem != nil {
			if option.Advanced {
				advancedForm.AddFormItem(formItem)
			} else {
				basicForm.AddFormItem(formItem)
			}
		}
	}

	if providerUpdate == nil && advancedForm.GetFormItemCount() > 0 {
		SetTabs(configWizardTabs + configWizardAdvancedTab)
	}

	App.SetFocus(basicForm)

	return nil
}

// updateButtons updates the buttons according to the page/form displayed.
//
//gocyclo:ignore
func (c *ConfigUI) updateButtons(page string) {
	if pg, _ := c.formUI.ManagerPages.GetFrontPage(); pg != page {
		return
	}

	switch page {
	case "manager":
		UpdateButtonView(c.formUI.ManagerButtons, func(label string) bool {
			if c.formUI.ManagerTable.GetRowCount() == 0 {
				return false
			}

			if c.formUI.ManagerTable.GetRowCount() <= 1 && label != "Create New" {
				return false
			}

			return true
		})

	case "wizard":
		UpdateButtonView(c.formUI.WizardButtons, func(label string) bool {
			page, _ := c.formUI.WizardPages.GetFrontPage()

			switch page {
			case "setup":
				if label == "Previous" || label == "Save" || label == "Filter" {
					return false
				}

			case "basic":
				if !HasTab("advanced") && label == "Next" {
					return false
				}

			case "advanced":
				if label == "Next" {
					return false
				}
			}

			return true
		})
	}
}

// updateHelp updates the help information for each item within the form.
func (c *ConfigUI) updateHelp(field string) {
	c.formUI.WizardHelp.Clear()

	options := []rclone.ProviderOption{
		{Name: "Name", Help: "Enter the name for this configuration."},
		{Name: "Type", Help: "Choose a provider type from the list."},
	}

	for _, option := range append(options, c.options...) {
		if option.Name == field && MatchProvider(option.Provider, c.getWizardData("provider")) {
			field = "[::bu]" + field + "[-:-:-]"
			if option.Provider != "" {
				field += tview.Escape(" [" + option.Provider + "]")
			}
			if option.Required {
				field += " (Required)"
			}

			c.formUI.WizardHelp.SetText(field + "\n" + option.Help)

			break
		}
	}

	c.formUI.WizardHelp.ScrollToBeginning()
}

// getWizardData returns the stored form data.
func (c *ConfigUI) getWizardData(key string) string {
	c.formUI.dataLock.RLock()
	defer c.formUI.dataLock.RUnlock()

	return modifyDataMap(c.formUI.WizardData, key, nil, false)
}

// setWizardData stores a form item's name(key) and its value.
func (c *ConfigUI) setWizardData(key string, value interface{}) {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	modifyDataMap(c.formUI.WizardData, strings.ToLower(key), value, true)
}

// clearWizardData clears the form data.
func (c *ConfigUI) clearWizardData() {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	c.options = nil
	c.formUI.WizardData = make(map[string]interface{})

	for _, form := range c.formUI.WizardForms {
		form.Clear(true)
	}
}

// checkWizardDataEmpty checks whether each specified field in
// the form's data is empty.
func (c *ConfigUI) checkWizardDataEmpty(fields ...string) error {
	for _, field := range fields {
		if c.getWizardData(field) == "" {
			return fmt.Errorf("Fill the %s field", field)
		}
	}

	return nil
}

// getWizardUpdating returns whether the form is being updated(true) or created(false).
func (c *ConfigUI) getWizardUpdating() bool {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	return c.WizardUpdating
}

// setWizardUpdating sets the form updation status.
func (c *ConfigUI) setWizardUpdating(updating bool) {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	c.WizardUpdating = updating
}

// getFormLoaded returns whether the form has loaded.
func (c *ConfigUI) getFormLoaded() bool {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	return c.formLoaded
}

// setFormLoaded sets the form loaded status
func (c *ConfigUI) setFormLoaded(loaded bool) {
	c.formUI.dataLock.Lock()
	defer c.formUI.dataLock.Unlock()

	c.formLoaded = loaded
}
