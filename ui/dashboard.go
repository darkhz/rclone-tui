package ui

import (
	"strconv"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/darkhz/tvxwidgets"
	"github.com/gdamore/tcell/v2"
)

// DashboardUI stores a layout to display dashboard information.
type DashboardUI struct {
	Table *tview.Table
	Plot  *tvxwidgets.Plot

	transfer [][]float64
}

var dashboard DashboardUI

// Name returns the page's name.
func (d *DashboardUI) Name() string {
	return "Dashboard"
}

// Focused returns the currently focused view.
func (d *DashboardUI) Focused() string {
	return d.Name()
}

// Init initializes the page.
func (d *DashboardUI) Init() bool {
	go d.populateDashboard(d.Table)

	return true
}

// Exit exits the page.
func (d *DashboardUI) Exit(page string) bool {
	rclone.StopDashboard()

	return true
}

// Layout returns this page's layout.
func (d *DashboardUI) Layout() tview.Primitive {
	d.Table = tview.NewTable()
	d.Table.SetBackgroundColor(tcell.ColorDefault)

	d.Plot = tvxwidgets.NewPlot()
	d.Plot.SetDrawAxes(false)
	d.Plot.SetLineColor([]tcell.Color{
		tcell.ColorBlue,
		tcell.ColorGreen,
	})
	d.Plot.SetMarker(tvxwidgets.PlotMarkerBraille)
	d.Plot.SetBackgroundColor(tcell.ColorDefault)

	d.transfer = [][]float64{{}, {}}

	plotInfo := tview.NewTextView()
	plotInfo.SetDynamicColors(true)
	plotInfo.SetTextAlign(tview.AlignRight)
	plotInfo.SetText("[blue::b]...[-:-:-] Speed [green::b]...[-:-:-] Average Speed")
	plotInfo.SetBackgroundColor(tcell.ColorDefault)

	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.Table, 0, 1, false).
		AddItem(plotInfo, 1, 0, false).
		AddItem(d.Plot, 0, 1, false)
}

// populateDashboard collects rclone stats and displays them.
func (d *DashboardUI) populateDashboard(t *tview.Table) {
	var gotInfo bool

	dashInfo, exit := rclone.StartDashboard()

	StartLoading("Loading stats")

	for {
		select {
		case <-exit:
			return

		case info := <-dashInfo:
			if !gotInfo && info.Connected {
				StopLoading()
				gotInfo = true
			}

			d.setDashboardInfo(info)
		}
	}
}

// setDashboardInfo sets the dashboard information.
func (d *DashboardUI) setDashboardInfo(info rclone.DashboardInfo) {
	connectStatus := "[green::b]Connected"
	if !info.Connected {
		connectStatus = "[red::b]Not Connected"
	}

	client, err := rclone.GetCurrentClient()
	if err != nil {
		return
	}

	layout := []struct {
		Title  string
		Info   string
		Header bool
	}{
		{"Overview", "", true},
		{"Status", connectStatus, false},
		{"Current URL", client.Hostname(), false},
		{"Bandwidth Control", info.Bandwidth, false},
		{"Version", info.Version, false},
		{},
		{"Global Stats", "", true},
		{"Running time", ReadableString(time.Duration(info.Stats.ElapsedTime) * time.Second), false},
		{"Average Speed", bytefmt.ByteSize(uint64(info.Stats.Speed)) + "/s", false},
		{"Transferred Bytes", bytefmt.ByteSize(uint64(info.Stats.Bytes)), false},
		{"Checks", strconv.FormatInt(info.Stats.Checks, 10), false},
		{"Deletes", strconv.FormatInt(info.Stats.Deletes, 10), false},
		{"Transfers", strconv.FormatInt(info.Stats.Transfers, 10), false},
		{"Errors", strconv.FormatInt(info.Stats.Errors, 10), false},
	}

	App.QueueUpdateDraw(func() {
		dashboard.Table.Clear()

		for i, stat := range layout {
			if !info.Connected && i > 1 {
				return
			}

			if info.Stats.Transferring != nil {
				for _, transfer := range info.Stats.Transferring {
					for i, speed := range []float64{
						transfer.Speed,
						transfer.SpeedAvg,
					} {
						if speed > 0 {
							d.transfer[i] = append(d.transfer[i], speed)
						}
					}
				}

				transferNorm := [][]float64{{}, {}}

				for i, data := range d.transfer {
					if len(data) > 100 {
						d.transfer[i] = d.transfer[i][len(d.transfer[i])-2:]
					}

					transferNorm[i] = Normalize(data...)
				}

				d.Plot.SetData(transferNorm)
			}

			title := stat.Title
			if title == "" {
				continue
			}
			if stat.Header {
				title = "[::bu]" + title
			} else {
				title = "[aqua::b]" + title
			}

			dashboard.Table.SetCell(i, 0, tview.NewTableCell(title).
				SetExpansion(1),
			)

			dashboard.Table.SetCell(i, 1, tview.NewTableCell("[::b]"+stat.Info))
		}
	})
}
