package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// JobUI stores a layout to display running job information.
type JobUI struct {
	View *tview.TreeView

	Lock sync.Mutex

	isOpen   bool
	prevPage string
}

var (
	jobUI JobUI

	jobIndicator = make(chan rclone.JobInfo, 10)
)

// JobMonitor monitors currently running jobs and displays them.
func JobMonitor() {
	for jobInfo := range rclone.JobInfoStatus() {
		select {
		case jobIndicator <- jobInfo:

		default:
		}

		select {
		case explorer.refreshPanes <- jobInfo:

		default:
		}

		if jobInfo.Error != "" {
			ErrorMessage("Job Monitor", fmt.Errorf(jobInfo.Error))
		}

		App.QueueUpdateDraw(func() {
			modifyJobNode(jobInfo)
		})
	}
}

// jobManager returns a display area for currently running jobs.
func jobManager() *tview.TreeView {
	if jobUI.View != nil {
		goto Layout
	}

	jobUI.View = tview.NewTreeView()
	jobUI.View.SetGraphics(false)
	jobUI.View.SetBackgroundColor(tcell.ColorDefault)
	jobUI.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeJobManager()

		case tcell.KeyCtrlX:
			node := jobUI.View.GetCurrentNode()
			if job, ok := node.GetReference().(*rclone.Job); ok {
				rclone.StopJobGroup(job)
			}
		}

		switch event.Rune() {
		case 'x':
			node := jobUI.View.GetCurrentNode()
			if job, ok := node.GetReference().(*rclone.Job); ok {
				job.Cancel()
			}
		}

		return event
	})

Layout:
	return jobUI.View
}

// modifyJobNode modifies the displayed job information for the given job.
func modifyJobNode(jobInfo rclone.JobInfo) {
	if jobUI.View == nil {
		return
	}

	if !isOpen() && !jobInfo.Finished {
		return
	}

	if strings.HasPrefix(jobInfo.Type, "UI:") {
		return
	}

	rootNode := jobUI.View.GetRoot()

	for i, jobTypeNode := range rootNode.GetChildren() {
		for _, jobNode := range jobTypeNode.GetChildren() {
			job := jobNode.GetReference().(*rclone.Job)

			if jobInfo.Group != "" {
				typeID := strings.Split(jobInfo.Group, "/")
				if len(typeID) < 2 {
					continue
				}

				if typeID[0] != job.Type ||
					typeID[1] != strconv.FormatInt(job.ID, 10) {
					continue
				}
			} else if job.ID != jobInfo.ID {
				continue
			}

			if jobInfo.Finished && jobInfo.Group == "" {
				jobTypeNode.RemoveChild(jobNode)

				if len(jobTypeNode.GetChildren()) == 0 {
					rootNode.RemoveChild(jobTypeNode)
					rootNode.RemoveChild(rootNode.GetChildren()[i])
				}

				continue
			}

			desc := "[::b]" + jobInfo.Description

			jobNode.SetText(desc)
			updateJobNodeDetails(jobNode, jobInfo)
		}
	}
}

// updateJobNodeDetails updates the information within the job node.
func updateJobNodeDetails(node *tview.TreeNode, jobInfo rclone.JobInfo) {
	if strings.Contains(jobInfo.Type, "Delete") {
		return
	}

	states := []string{
		"Percentage: ",
		"ETA: ",
	}

	transferStats := jobInfo.CurrentTransfer
	if transferStats.Percentage == 0 && transferStats.Bytes > 0 {
		states[0] = "Transferred: "
		states[0] += bytefmt.ByteSize(uint64(transferStats.Bytes))
	} else {
		states[0] += strconv.FormatInt(transferStats.Percentage, 10) + "%"
	}

	if transferStats.Speed > 0 {
		states[0] += " (" + bytefmt.ByteSize(uint64(transferStats.Speed)) + "/s)"
	}

	if transferStats.Eta == 0 {
		states[1] += "Unspecified"
	} else {
		states[1] += ReadableString(time.Duration(float64(transferStats.Eta)) * time.Second)
	}

	if len(node.GetChildren()) == 0 {
		for i := 0; i < len(states); i++ {
			node.AddChild(tview.NewTreeNode(""))
		}
	}

	for j, node := range node.GetChildren() {
		node.SetText(states[j])
	}
}

// openJobManager displays the job manager.
func openJobManager() {
	var rootNode *tview.TreeNode

	if jobUI.View != nil {
		rootNode = jobUI.View.GetRoot()
		rootNode.ClearChildren()

		goto SwitchToView
	}

	rootNode = tview.NewTreeNode("[::bu]Job Manager")
	rootNode.SetSelectable(false)

SwitchToView:
	rootNode.AddChild(
		tview.NewTreeNode("").SetSelectable(false),
	)

	rclone.GetJobQueue().Range(func(key, value interface{}) bool {
		jobType := key.(string)
		if strings.HasPrefix(jobType, "UI:") {
			return true
		}

		jobMap := value.(map[int64]*rclone.Job)

		jobTypeNode := tview.NewTreeNode("[::b]- [::bu]" + jobType)
		jobTypeNode.SetSelectable(false)
		jobTypeNode.SetColor(tcell.ColorPurple)

		for _, job := range jobMap {
			jobNode := tview.NewTreeNode("[::b]" + job.Description)
			jobNode.SetReference(job)
			jobNode.SetColor(tcell.ColorGreen)

			jobTypeNode.AddChild(jobNode)
		}

		rootNode.AddChild(jobTypeNode).AddChild(
			tview.NewTreeNode("").SetSelectable(false),
		)

		return true
	})

	jobUI.prevPage, _ = MainPage.GetFrontPage()
	MainPage.AddAndSwitchToPage("job_view", jobManager(), true)

	jobUI.View.SetRoot(rootNode)
	if rootChildren := rootNode.GetChildren(); len(rootChildren) > 0 {
		jobUI.View.SetCurrentNode(rootChildren[len(rootChildren)-1])
	}

	setOpen(true)
}

// closeJobManager closes the job manager.
func closeJobManager() {
	MainPage.SwitchToPage(jobUI.prevPage)

	setOpen(false)
}

// isOpen checks whether the job manager is displayed.
func isOpen() bool {
	jobUI.Lock.Lock()
	defer jobUI.Lock.Unlock()

	return jobUI.isOpen
}

// setOpen sets the job manager display status.
func setOpen(open bool) {
	jobUI.Lock.Lock()
	defer jobUI.Lock.Unlock()

	jobUI.isOpen = open
}
