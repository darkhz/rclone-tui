package rclone

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"code.cloudfoundry.org/bytefmt"
	"github.com/darkhz/rclone-tui/rclone"
)

// Mkdir creates a directory within the remote.
func Mkdir(id, fs, remote, name string) error {
	command := map[string]interface{}{
		"fs":     fs,
		"remote": filepath.Join(remote, name),
	}

	job, err := rclone.SendCommandAsync(
		"UI:Explorer:"+id, "Creating directory "+name,
		command, "/operations/mkdir", struct{}{},
	)
	if err != nil {
		return err
	}

	go rclone.MonitorJob(job, struct{}{})

	jobInfo, err := rclone.GetJobReply(job)
	if err != nil {
		return err
	}

	listItem, err := stat(rclone.GetClientContext(), fs, command["remote"].(string))
	if err != nil {
		return err
	}

	listItem = appendItemDetails(listItem, fs)
	listItem.RefreshAddItem = true
	job.RefreshItems = []ListItem{listItem}

	rclone.StopJob(job, jobInfo.Error)

	return nil
}

// PublicLink returns a public link for the provided item.
func PublicLink(id, fs, remote string, item ListItem) (string, error) {
	command := map[string]interface{}{
		"fs":     fs,
		"remote": filepath.Join(remote, item.Name),
	}

	job, err := rclone.SendCommandAsync("UI:Explorer:"+id, "Generating public link", command, "/operations/publiclink")
	if err != nil {
		return "", err
	}

	jobInfo, err := rclone.GetJobReply(job)
	if err != nil {
		return "", err
	}

	url, ok := jobInfo.Output["url"]
	if !ok || ok && url == nil {
		return "", fmt.Errorf("Public link could not be generated")
	}

	return url.(string), nil
}

// Copy copies a list of items to the destination remote and path.
func Copy(items []ListItem, dstFs, dstRemote string) {
	BatchOperation(
		"Copy", "Copying", dstFs, dstRemote,
		[]string{"/sync/copy", "/operations/copyfile"}, items,
	)
}

// Move moves a list of items to the destination remote and path.
func Move(items []ListItem, dstFs, dstRemote string) {
	BatchOperation(
		"Move", "Moving", dstFs, dstRemote,
		[]string{"/sync/move", "/operations/movefile"}, items,
	)
}

// Delete deletes a list of items from the remote.
func Delete(items []ListItem) {
	BatchOperation(
		"Delete", "Deleting", "", "",
		[]string{"/operations/purge", "/operations/deletefile"}, items,
	)
}

// BatchOperation starts a batch job on a list of items.
//
//gocyclo:ignore
func BatchOperation(
	name, desc, dstFs, dstRemote string, endpoints []string, items []ListItem,
) {
	if items == nil {
		return
	}

	id := rclone.GetNewJobID(name)

	job := rclone.NewJob(
		name, desc, id,
		name+"/"+strconv.FormatInt(id, 10),
	)

	rclone.AddJobToQueue(job, struct{}{})

	go func(mainJob *rclone.Job) {
		var jobErr string

		for i, item := range items {
			var endpoint string
			var description string

			select {
			case <-mainJob.Context.Done():
				break

			default:
			}

			if item.IsDir {
				endpoint = endpoints[0]
			} else {
				endpoint = endpoints[1]
			}

			command := batchCommand(name, dstFs, dstRemote, item)
			command["_group"] = mainJob.Group

			description += "(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(items)) + ") "
			description += desc + " " + filepath.Base(item.Path)
			if desc != "Deleting" {
				description += " -> " + dstFs + dstRemote
			}

			job, err := rclone.SendCommandAsync(
				"_"+name, description,
				command, endpoint, struct{}{},
			)
			if err != nil {
				jobErr = err.Error()
				break
			}

			job.Group = mainJob.Group
			job.Context = mainJob.Context
			job.Cancel = mainJob.Cancel

			go rclone.MonitorJob(job, struct{}{})

			jobInfo, err := rclone.GetJobReply(job)
			if err != nil || jobInfo.Error != "" {
				break
			}

			refreshItems := []ListItem{}

			if name == "Delete" || name == "Move" {
				item.RefreshAddItem = false
				refreshItems = append(refreshItems, item)
			}

			if name == "Copy" || name == "Move" {
				item.FS = dstFs
				item.Path = filepath.Join(dstRemote, item.Name)
				item.RefreshAddItem = true

				if item.Size == -1 {
					listItem, err := stat(job.Context, item.FS, item.Path)
					if err != nil {
						continue
					}

					item.Size = listItem.Size
					if listItem.Size > 0 {
						item.ISize = bytefmt.ByteSize(uint64(item.Size))
					}
				}

				refreshItems = append(refreshItems, item)
			}

			job.RefreshItems = refreshItems

			rclone.StopJob(job, jobInfo.Error)
		}

		rclone.StopJob(mainJob, jobErr, struct{}{})
	}(job)
}

// stat returns the information for the item.
func stat(ctx context.Context, fs, remote string) (ListItem, error) {
	var listItem ListItem

	item := struct {
		Item ListItem `json:"item"`
	}{}

	command := map[string]interface{}{
		"fs":     fs,
		"remote": remote,
	}

	response, err := rclone.SendCommand(command, "/operations/stat", ctx)
	if err == nil {
		err = response.Decode(&item)
		listItem = item.Item
	}

	return listItem, err
}

// batchCommand returns a command in an rclone-parseable format.
func batchCommand(operation, dstFs, dstRemote string, item ListItem) map[string]interface{} {
	var command map[string]interface{}

	switch operation {
	case "Copy", "Move":
		if item.IsDir {
			command = map[string]interface{}{
				"srcFs": item.Path,
				"dstFs": dstFs + filepath.Join(dstRemote, item.Name),
			}
		} else {
			command = map[string]interface{}{
				"srcFs":     item.FS,
				"srcRemote": item.Path,
				"dstFs":     dstFs,
				"dstRemote": filepath.Join(dstRemote, item.Name),
			}
		}

	case "Delete":
		command = map[string]interface{}{
			"fs":     item.FS,
			"remote": item.Path,
		}
	}

	return command
}
