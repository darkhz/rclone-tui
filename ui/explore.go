package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/darkhz/rclone-tui/rclone"
	rcfns "github.com/darkhz/rclone-tui/rclone/operations"
	"github.com/darkhz/tview"
	"github.com/darkhz/tvxwidgets"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// ExplorerUI stores the layout for the explorer page.
type ExplorerUI struct {
	Panes []*Pane
	Flex  *tview.Flex

	currentPane int
	dataLock    sync.Mutex

	selections    map[rcfns.ListItem]struct{}
	selectionLock sync.Mutex

	init         bool
	numPanes     int
	refreshPanes chan rclone.JobInfo
}

// Pane stores the layout for a single explorer pane.
type Pane struct {
	ID, FS, Path string

	Title *tview.TextView
	View  *tview.Table

	Flex  *tview.Flex
	Modal *Modal

	Status        *tview.Pages
	Input         *tview.InputField
	AboutText     *tview.TextView
	SortListText  *tview.TextView
	SpinnerText   *tview.TextView
	Spinner       *tvxwidgets.Spinner
	SpinnerCancel chan struct{}

	Lock *semaphore.Weighted

	list       rcfns.List
	savedPaths map[string]string

	sortMode string
	sortAsc  bool

	aboutCtx     context.Context
	remoteCtx    context.Context
	aboutCancel  context.CancelFunc
	remoteCancel context.CancelFunc

	filtered, isloading bool
	refreshChan         chan rclone.JobInfo
}

var explorer ExplorerUI

// Name returns the page's name.
func (e *ExplorerUI) Name() string {
	return "Explorer"
}

// Focused returns the currently focused view.
func (e *ExplorerUI) Focused() string {
	return e.Name() + ":" + e.getPane().ID
}

// Init initializes the page.
func (e *ExplorerUI) Init() bool {
	if e.init {
		return true
	}

	e.init = true
	go e.getPane().List()

	return e.init
}

// Exit exits the page.
func (e *ExplorerUI) Exit(page string) bool {
	e.getPane().remoteCancel()

	return true
}

// Layout returns this page's layout.
func (e *ExplorerUI) Layout() tview.Primitive {
	e.refreshPanes = make(chan rclone.JobInfo, 10)
	e.selections = make(map[rcfns.ListItem]struct{})

	e.Flex = tview.NewFlex()
	e.Flex.SetDirection(tview.FlexColumn)
	e.Flex.SetFocusFunc(func() {
		App.SetFocus(e.getPane().View)
	})

	if e.numPanes == 0 {
		e.numPanes = 2
	}

	go e.detectPaneRefresh()

	for i := 0; i < e.numPanes; i++ {
		title := tview.NewTextView()
		title.SetDynamicColors(true)
		title.SetTextAlign(tview.AlignCenter)
		title.SetText("Press 'g' to select remote")
		title.SetBackgroundColor(tcell.ColorDefault)

		view := tview.NewTable()
		view.SetBorder(true)
		view.SetSelectorWrap(true)
		view.SetFocusBorder(false)
		view.SetSelectable(true, false)
		view.SetBackgroundColor(tcell.ColorDefault)
		view.SetFocusFunc(func() {
			view.SetBorderColor(tcell.ColorBlue)
		})
		view.SetBlurFunc(func() {
			view.SetBorderColor(tcell.ColorWhite)
		})
		view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCtrlR:
				go e.getPane().List()

			case tcell.KeyCtrlX:
				e.getPane().remoteCancel()

			case tcell.KeyEscape:
				go e.reloadPanes(!e.getPane().filtered)

			case tcell.KeyTab:
				e.switchPanes(event.Key() == tcell.KeyBacktab)

			case tcell.KeyLeft, tcell.KeyRight:
				e.getPane().ChangeDir(event.Key() == tcell.KeyLeft)
			}

			switch event.Rune() {
			case 'g':
				go e.getPane().ShowRemotes()

			case '/':
				e.getPane().Filter()

			case ',':
				e.getPane().Sort()

			case 'p', 'm', 'd', 'M', ';', 'i':
				go e.getPane().Operation(event.Rune())

			case ' ', 'a', 'A':
				e.getPane().Select(event.Rune() == 'A', event.Rune() == 'a')
			}

			return event
		})

		input := tview.NewInputField()
		input.SetLabelColor(tcell.ColorWhite)
		input.SetBackgroundColor(tcell.ColorDefault)
		input.SetFieldBackgroundColor(tcell.ColorDefault)

		spinner := tvxwidgets.NewSpinner()
		spinner.SetCustomStyle(nil)
		spinner.SetBackgroundColor(tcell.ColorDefault)

		spinnerText := tview.NewTextView()
		spinnerText.SetDynamicColors(true)
		spinnerText.SetBackgroundColor(tcell.ColorDefault)

		aboutText := tview.NewTextView()
		aboutText.SetDynamicColors(true)
		aboutText.SetBackgroundColor(tcell.ColorDefault)

		sortListText := tview.NewTextView()
		sortListText.SetRegions(true)
		sortListText.SetDynamicColors(true)
		sortListText.SetBackgroundColor(tcell.ColorDefault)

		spinnerCancel := make(chan struct{})

		spinnerFlex := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(spinner, 1, 0, false).
			AddItem(nil, 1, 0, false).
			AddItem(spinnerText, 0, 1, false)

		status := tview.NewPages()
		status.AddPage("about", aboutText, true, false)
		status.AddPage("input", input, true, false)
		status.AddPage("sort", sortListText, true, false)
		status.AddPage("load", spinnerFlex, true, false)
		status.SetBackgroundColor(tcell.ColorDefault)

		layout := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(title, 1, 0, false).
			AddItem(view, 0, 1, true).
			AddItem(status, 1, 0, false)
		layout.SetBackgroundColor(tcell.ColorDefault)

		lock := semaphore.NewWeighted(1)

		pane := Pane{
			ID: strconv.Itoa(i),

			Title: title,
			View:  view,
			Flex:  layout,

			Status:        status,
			Input:         input,
			AboutText:     aboutText,
			SortListText:  sortListText,
			Spinner:       spinner,
			SpinnerText:   spinnerText,
			SpinnerCancel: spinnerCancel,

			Lock: lock,

			sortMode: "name",
			sortAsc:  true,

			savedPaths:  make(map[string]string),
			refreshChan: make(chan rclone.JobInfo, 10),
		}

		e.Panes = append(e.Panes, &pane)

		e.Flex.AddItem(pane.Flex, 0, 1, true)

		go pane.watchItem()
	}

	return e.Flex
}

// List lists the directory items for the current remote and path.
func (p *Pane) List(item ...rcfns.ListItem) {
	if !p.Lock.TryAcquire(1) {
		return
	}
	defer p.Lock.Release(1)

	var listItem rcfns.ListItem

	if item != nil {
		listItem = item[0]
		goto StartListing
	}

	if p.FS == "" {
		p.ShowRemotes()
		return
	}

	listItem.FS = p.FS
	listItem.Path = p.Path

StartListing:
	App.QueueUpdateDraw(func() {
		p.View.SetSelectable(false, false)
	})
	defer App.QueueUpdateDraw(func() {
		p.View.SetSelectable(true, false)

		if page, _ := MainPage.GetFrontPage(); page == "explorer" {
			App.SetFocus(explorer.getPane().View)
		}
	})

	go p.startLoading("Listing " + listItem.FS + listItem.Path)
	defer p.stopLoading()

	list, err := rcfns.ListFS(p.ID, listItem.FS, listItem.Path)
	if err != nil {
		ErrorMessage("Explorer", err)
		return
	}
	sortList(list.Items, p.sortAsc, p.sortMode)

	p.list = list
	p.FS = listItem.FS
	p.Path = listItem.Path

	p.filtered = false

	if listItem.About {
		go p.setAbout()
	}

	App.QueueUpdateDraw(func() {
		p.viewList(list)
	})
}

// ChangeDir changes the current directory.
func (p *Pane) ChangeDir(cdback bool) {
	var path, dir string
	var listItem rcfns.ListItem

	if !cdback {
		_, list, err := p.getSelection()
		if err != nil {
			return
		}

		if !list.IsDir {
			return
		}

		path = list.Path
		dir = filepath.Base(list.Name)

		_, dir = rcfns.GetListPath(path, dir, cdback)
		list.Name = filepath.Base(dir)

		listItem = list
	} else {
		path, _ = rcfns.GetListPath(p.Path, "", cdback)
		listItem.Path = path
	}

	listItem.FS = p.FS

	go p.List(listItem)
}

// ShowRemotes displays a modal with a list of available remotes.
func (p *Pane) ShowRemotes() {
	InfoMessage("Getting remotes...", true)
	defer InfoMessage("", false)

	if p.remoteCtx != nil {
		p.remoteCancel()
	}

	p.remoteCtx, p.remoteCancel = context.WithCancel(context.Background())

	remotes, err := rcfns.ListRemotes(p.remoteCtx)
	if err != nil {
		ErrorMessage("Explorer", err)
		return
	}

	remotes = append(remotes, "local")

	p.Modal = NewModal("show_remotes", "Select remote", true, false, len(remotes)+6, 60)
	remoteInput, remoteTable := p.Modal.Input, p.Modal.Table

	updateTable := func(text ...string) {
		var row int

		remoteTable.Clear()

		for _, remote := range remotes {
			if text != nil && strings.Index(remote, text[0]) == -1 {
				continue
			}

			remoteTable.SetCell(row, 0, tview.NewTableCell(tview.Escape(remote)).
				SetExpansion(1).
				SetReference(remote).
				SetAlign(tview.AlignCenter),
			)

			row++
		}
	}

	remoteTable.SetSelectorWrap(false)

	remoteInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown, tcell.KeyUp:
			remoteTable.InputHandler()(event, nil)

		case tcell.KeyEscape:
			p.Modal.Exit()

		case tcell.KeyEnter:
			var fs, path string

			row, _ := remoteTable.GetSelection()
			remote, ok := remoteTable.GetCell(row, 0).
				GetReference().(string)
			if !ok {
				goto Event
			}

			p.savedPaths[p.FS] = p.Path

			if remote == "local" {
				fs = "/"
			} else {
				fs = remote + ":"
			}

			path = p.savedPaths[fs]
			p.Modal.Exit()

			p.remoteCtx, p.remoteCancel = context.WithCancel(context.Background())

			go func() {
				if job, err := rclone.GetLatestJob("UI:" + explorer.Focused()); err == nil {
					if strings.Contains(job.Description, "Listing") {
						job.Cancel()

						p.Lock.Acquire(p.remoteCtx, 1)
						p.Lock.Release(1)
					}
				}

				go p.List(rcfns.ListItem{FS: fs, Path: path, About: true})
			}()
		}

	Event:
		return event
	})
	remoteInput.SetChangedFunc(func(text string) {
		updateTable(text)
	})

	updateTable()

	App.QueueUpdateDraw(func() {
		p.Modal.Show()
	})
}

// Operation executes an operation according to the key pressed.
//
//gocyclo:ignore
func (p *Pane) Operation(key rune) {
	switch key {
	case 'p':
		rcfns.Copy(explorer.getSelectionsList(), p.FS, p.Path)

	case 'm':
		rcfns.Move(explorer.getSelectionsList(), p.FS, p.Path)

	case 'd':
		list := explorer.getSelectionsList()
		if len(list) == 0 {
			return
		}

		if !ConfirmInput("Delete selected files? (y/n)") {
			return
		}

		rcfns.Delete(explorer.getSelectionsList())

	case 'M':
		dirName := SetInput("Create directory:", struct{}{})
		fullPath := p.FS + filepath.Join(p.Path, dirName)

		go p.startLoading("Creating " + fullPath)
		defer p.stopLoading()

		if err := rcfns.Mkdir(p.ID, p.FS, p.Path, dirName); err != nil {
			ErrorMessage("Explorer", err)
		}

		go p.List()

		return

	case ';':
		_, item, err := p.getSelection()
		if err != nil {
			return
		}

		go p.startLoading("Loading public link for " + item.Name)
		defer p.stopLoading()

		publiclink, err := rcfns.PublicLink(p.ID, p.FS, p.Path, item)
		if err != nil {
			ErrorMessage("Explorer", err, struct{}{})
			return
		}

		modal := NewModal("public_link", "Public Link", false, true, 10, len(publiclink)+10)

		modal.TextView.SetText(publiclink)
		modal.TextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				modal.Exit()
			}

			return event
		})

		modal.Show()

	case 'i':
		if p.FS == "" {
			return
		}

		go p.startLoading("Loading fs information for " + p.FS)
		defer p.stopLoading()

		fsinfo, err := rcfns.FsInfo(p.ID, p.FS)
		if err != nil {
			ErrorMessage("Explorer", err)
			return
		}

		modal := NewModal("fsinfo", "FS Information", false, true, 60, 100)
		modal.TextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape:
				modal.Exit()
			}

			return event
		})

		App.QueueUpdateDraw(func() {
			for _, detail := range [][]string{
				{"Name", fsinfo.Name},
				{"Root", fsinfo.Root},
				{"Log String", fsinfo.String},
				{"Hashes"},
				{"Features"},
			} {
				if detail[0] == "Hashes" {
					detail = append(detail, fsinfo.Hashes...)
				} else if detail[0] == "Features" {
					detail = append(detail, fsinfo.FeatureList...)
				}

				if len(detail) > 2 {
					fmt.Fprintf(modal.TextView, "\n[::bu]%s[-:-:-]\n", detail[0])

					for _, d := range detail[2:] {
						fmt.Fprintf(modal.TextView, "%s\n", d)
					}

					continue
				}

				fmt.Fprintf(modal.TextView, "[::bu]%s[-:-:-]: %s\n", detail[0], detail[1])
			}

			modal.Show()
		})
	}
}

// Select selects multiple items within the current directory. The selected
// items can then be used within an operation, for example copying files.
func (p *Pane) Select(all, inverse bool) {
	if !p.Lock.TryAcquire(1) {
		return
	}
	defer p.Lock.Release(1)

	row, listItem, err := p.getSelection()
	if err != nil {
		return
	}

	if all {
		for _, item := range p.list.Items {
			p.itemSelected(item, true)
		}

		p.viewList(p.list)

		p.View.Select(row, 0)

		return
	}

	if inverse {
		for row, item := range p.list.Items {
			p.viewList(rcfns.List{
				Items: []rcfns.ListItem{item},
			}, row)
		}

		p.View.Select(row, 0)

		return
	}

	p.viewList(rcfns.List{
		Items: []rcfns.ListItem{listItem},
	}, row)
}

// Filter filters the items within the current directory.
func (p *Pane) Filter() {
	if !p.Lock.TryAcquire(1) {
		return
	}
	defer p.Lock.Release(1)

	p.Status.SwitchToPage("input")

	p.Input.SetText("")
	p.Input.SetLabel("[::b]Filter: ")
	p.Input.SetChangedFunc(func(text string) {
		var items []rcfns.ListItem

		for _, item := range p.list.Items {
			if strings.Index(
				strings.ToLower(item.Name),
				strings.ToLower(text),
			) != -1 {
				items = append(items, item)
			}
		}

		p.viewList(rcfns.List{Items: items})
	})
	p.Input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if p.isloading {
				p.Status.SwitchToPage("load")
			} else {
				p.Status.SwitchToPage("about")
			}

			p.filtered = true

			App.SetFocus(p.Flex)
		}

		return event
	})

	App.SetFocus(p.Input)
}

// Sort sorts the current directory list.
func (p *Pane) Sort() {
	if !p.Lock.TryAcquire(1) {
		return
	}
	defer p.Lock.Release(1)

	var lastRune rune

	arrange := func() string {
		var sortMethod string

		if p.sortAsc {
			sortMethod = " asc "
		} else {
			sortMethod = " desc "
		}

		return "[::b]Sort by[-:-:-]" + sortMethod
	}

	text := `["name"](n)ame[""] ["size"](s)ize[""] ["modified"](m)odified[""]`

	p.Status.SwitchToPage("sort")

	p.SortListText.SetText(arrange() + text)
	p.SortListText.Highlight(p.sortMode)
	p.SortListText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if p.isloading {
				p.Status.SwitchToPage("load")
			} else {
				p.Status.SwitchToPage("about")
			}

			App.SetFocus(p.Flex)

		case tcell.KeyRune:
			for _, region := range p.SortListText.GetRegionIDs() {
				if rune(region[0]) != event.Rune() {
					continue
				}

				p.sortMode = region

				if lastRune == event.Rune() {
					p.sortAsc = !p.sortAsc
				} else {
					p.sortAsc = true
				}
				lastRune = event.Rune()

				p.SortListText.SetText(arrange() + text)
				p.SortListText.Highlight(region)

				sortList(p.list.Items, p.sortAsc, p.sortMode)
				p.viewList(p.list)

				return nil
			}
		}

		return event
	})

	App.SetFocus(p.SortListText)
}

// viewList displays the current directory listing in the pane.
func (p *Pane) viewList(list rcfns.List, selectRow ...int) {
	if selectRow != nil {
		goto ShowList
	}

	p.View.Clear()

	if title := p.FS + p.Path; title != "" {
		p.Title.SetText("[::bu]" + tview.Escape(title))
		p.Title.ScrollToEnd()
	}

ShowList:
	for row, item := range list.Items {
		var itemColor, infoColor tcell.Color

		if selectRow != nil {
			row = selectRow[0]
		}

		name := item.Name
		if item.IsDir {
			name += "/"
		}

		if ok := p.itemSelected(item, false, selectRow...); ok {
			itemColor = tcell.ColorOrange
			infoColor = tcell.ColorOrange
		} else {
			if item.IsDir {
				itemColor = tcell.ColorBlue
			} else {
				itemColor = tcell.ColorWhite
			}

			infoColor = tcell.ColorGrey
		}

		p.View.SetCell(row, 0, tview.NewTableCell(item.ISize).
			SetReference(item).
			SetTextColor(infoColor).
			SetBackgroundColor(tcell.ColorDefault).
			SetSelectedStyle(tcell.Style{}.
				Bold(true),
			),
		)
		p.View.SetCell(row, 1, tview.NewTableCell(item.ModifiedTime).
			SetReference(item).
			SetTextColor(infoColor).
			SetBackgroundColor(tcell.ColorDefault).
			SetSelectedStyle(tcell.Style{}.
				Bold(true),
			),
		)
		p.View.SetCell(row, 2, tview.NewTableCell(tview.Escape(name)).
			SetExpansion(1).
			SetTextColor(itemColor).
			SetAttributes(tcell.AttrBold).
			SetBackgroundColor(tcell.ColorDefault).
			SetSelectedStyle(tcell.Style{}.
				Foreground(itemColor).
				Background(tcell.Color16),
			),
		)

		if selectRow != nil {
			if rowCount := p.View.GetRowCount(); row >= rowCount-1 {
				row = rowCount - 1
			} else {
				row++
			}

			p.View.Select(row, 0)

			return
		}
	}

	p.View.ScrollToBeginning()

	p.View.Select(0, 0)
}

// watchItem watches for whether items in the current directory
// have been added or removed.
func (p *Pane) watchItem() {
	for info := range p.refreshChan {
		if !info.Finished || info.Error != "" {
			continue
		}

		items, ok := info.RefreshItems.([]rcfns.ListItem)
		if !ok {
			continue
		}

		func() {
			p.Lock.Acquire(context.Background(), 1)
			defer p.Lock.Release(1)

			for _, item := range items {
				var exist bool

				itemPath := filepath.Dir(item.Path)
				if itemPath == "." {
					itemPath = ""
				}

				if p.FS+p.Path != item.FS+itemPath {
					continue
				}

				for i, listItem := range p.list.Items {
					if listItem.ID == item.ID || listItem.Name == item.Name {
						if item.RefreshAddItem {
							p.list.Items[i].Size = item.Size
							p.list.Items[i].ISize = item.ISize

							exist = true
						} else {
							list := p.list.Items
							list = list[:i+copy(list[i:], list[i+1:])]

							p.list.Items = list
						}

						break
					}
				}

				if !exist && item.RefreshAddItem {
					p.list.Items = append(p.list.Items, item)
					sortList(p.list.Items, p.sortAsc, p.sortMode)
				}

				App.QueueUpdateDraw(func() {
					p.viewList(p.list)
				})
			}
		}()
	}
}

// getSelection returns the current directory item selection.
func (p *Pane) getSelection() (int, rcfns.ListItem, error) {
	row, _ := p.View.GetSelection()

	cell := p.View.GetCell(row, 0)
	list, ok := cell.GetReference().(rcfns.ListItem)
	if !ok {
		return -1, rcfns.ListItem{}, fmt.Errorf("Cannot select list item")
	}

	return row, list, nil
}

// itemSelected returns whether an item is selected. If modify is set, it will
// modify the item's selected status.
func (p *Pane) itemSelected(key rcfns.ListItem, forceSelect bool, modify ...int) bool {
	explorer.selectionLock.Lock()
	defer explorer.selectionLock.Unlock()

	_, ok := explorer.selections[key]

	if modify != nil || forceSelect {
		if forceSelect || (!ok && modify != nil) {
			explorer.selections[key] = struct{}{}
		} else if ok && modify != nil {
			delete(explorer.selections, key)
		}

		ok = !ok
	}

	return ok
}

// setAbout sets the space information for a remote.
func (p *Pane) setAbout() {
	var aboutText string

	if p.aboutCtx != nil {
		p.aboutCancel()
	}

	p.aboutCtx, p.aboutCancel = context.WithCancel(context.Background())

	about, err := rcfns.AboutFS(p.aboutCtx, p.FS)
	if err != nil {
		return
	}

	if about.Total <= 0 {
		return
	}

	aboutText += bytefmt.ByteSize(uint64(about.Used)) + "/" +
		bytefmt.ByteSize(uint64(about.Total)) + " used, " +
		bytefmt.ByteSize(uint64(about.Free)) + " free"

	App.QueueUpdateDraw(func() {
		p.AboutText.SetText(" " + aboutText)
	})
}

// startLoading starts the loading spinner for a pane.
func (p *Pane) startLoading(message string) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	App.QueueUpdateDraw(func() {
		if pg, _ := p.Status.GetFrontPage(); pg != "input" {
			p.Status.SwitchToPage("load")
		}

		p.SpinnerText.SetText("[yellow::b]" + message)
		p.Spinner.SetStyle(tvxwidgets.SpinnerDotsCircling)

		p.isloading = true
	})

	for {
		select {
		case <-t.C:
			App.QueueUpdateDraw(func() {
				p.Spinner.Pulse()
			})

		case <-p.SpinnerCancel:
			App.QueueUpdateDraw(func() {
				if pg, _ := p.Status.GetFrontPage(); pg != "input" {
					p.Status.SwitchToPage("about")
				}

				p.SpinnerText.SetText("")
				p.Spinner.SetCustomStyle(nil)

				p.isloading = false
			})

			return
		}
	}
}

// stopLoading stops the loading spinner for a pane.
func (p *Pane) stopLoading() {
	p.SpinnerCancel <- struct{}{}
}

// getSelectionsList returns the list of all the selected items across
// all panes.
func (e *ExplorerUI) getSelectionsList() []rcfns.ListItem {
	e.selectionLock.Lock()
	defer e.selectionLock.Unlock()

	var list []rcfns.ListItem

	for selection := range e.selections {
		list = append(list, selection)
	}

	return list
}

// getPane returns the currently focused pane.
func (e *ExplorerUI) getPane() *Pane {
	return explorer.Panes[explorer.currentPane]
}

// reloadPanes clears all selections across all panes.
func (e *ExplorerUI) reloadPanes(clearSelections bool) {
	if clearSelections {
		e.selectionLock.Lock()
		e.selections = make(map[rcfns.ListItem]struct{})
		e.selectionLock.Unlock()
	}

	e.dataLock.Lock()
	defer e.dataLock.Unlock()

	for _, p := range e.Panes {
		if !p.Lock.TryAcquire(1) {
			continue
		}
		defer p.Lock.Release(1)

		if p.FS == "" {
			continue
		}

		App.QueueUpdateDraw(func() {
			p.viewList(p.list)
		})

		p.filtered = false
	}
}

// switchPanes cycles between the panes.
func (e *ExplorerUI) switchPanes(reverse bool) {
	e.dataLock.Lock()
	defer e.dataLock.Unlock()

	currentPane := e.currentPane

	prevpane := e.Panes[currentPane]
	prevpath := prevpane.FS + prevpane.Path

	if reverse {
		currentPane--
	} else {
		currentPane++
	}

	if currentPane >= len(e.Panes) {
		currentPane = 0
	} else if currentPane < 0 {
		currentPane = len(e.Panes) - 1
	}

	currentpane := e.Panes[currentPane]
	currentpath := currentpane.FS + currentpane.Path

	if prevpath == currentpath {
		go App.QueueUpdateDraw(func() {
			row, _, _ := currentpane.getSelection()

			currentpane.viewList(currentpane.list)
			currentpane.View.Select(row, 0)

		})
	}

	App.SetFocus(currentpane.View)

	e.currentPane = currentPane
}

// detectPaneRefresh gets a signal and sends it to all the panes
// so as to enable refreshing their current content.
func (e *ExplorerUI) detectPaneRefresh() {
	for jobInfo := range e.refreshPanes {
		e.dataLock.Lock()

		for _, pane := range e.Panes {
			select {
			case pane.refreshChan <- jobInfo:

			default:
			}
		}

		e.dataLock.Unlock()
	}
}
