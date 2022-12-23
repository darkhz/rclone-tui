package rclone

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/mitchellh/mapstructure"
)

// List stores a list of directory entries.
type List struct {
	Items []ListItem `mapstructure:"list"`

	Path string
}

// ListItem stores information about a directory entry.
type ListItem struct {
	ID       string `mapstructure:"ID"`
	IsDir    bool   `mapstructure:"IsDir"`
	MimeType string `mapstructure:"MimeType"`
	ModTime  string `mapstructure:"ModTime"`
	Name     string `mapstructure:"Name"`
	Path     string `mapstructure:"Path"`
	Size     int64  `mapstructure:"Size"`

	FS               string
	ISize            string
	ModifiedTime     string
	ModifiedTimeUnix int64

	About          bool
	RefreshAddItem bool
}

// ListFS returns a list of directory entries from the provided remote and path.
func ListFS(id, fstype, path string) (List, error) {
	var list List
	var fs, desc string

	if strings.Contains(fstype, ":") {
		fs = fstype
		if path != "" {
			path = filepath.Clean(path)
		}
	} else {
		fs = fstype
	}

	desc = fs + path

	command := map[string]interface{}{
		"fs":     fs,
		"remote": path,
	}

	job, err := rclone.SendCommandAsync("UI:Explorer:"+id, "Listing "+desc, command, "/operations/list")
	if err != nil {
		return List{}, err
	}

	jobInfo, err := rclone.GetJobReply(job)
	if err != nil {
		return List{}, err
	}

	err = mapstructure.Decode(jobInfo.Output, &list)
	if err != nil {
		return List{}, err
	}

	for j, item := range list.Items {
		list.Items[j] = appendItemDetails(item, fs)
	}

	return list, err
}

func appendItemDetails(item ListItem, fs string) ListItem {
	modtime, _ := time.Parse(time.RFC3339, item.ModTime)

	item.ModTime = ""
	item.ModifiedTimeUnix = modtime.Unix()
	item.ModifiedTime = modtime.Format("Mon 01/02 15:04")

	size := "Unknown"
	if item.Size >= 0 {
		size = bytefmt.ByteSize(uint64(item.Size))
	}

	item.ISize = size

	item.FS = fs

	return item
}

// ListRemotes lists the configured remotes.
func ListRemotes(ctx context.Context) ([]string, error) {
	return GetDataSlice(ctx, "/config/listremotes", "remotes")
}

// GetListPath returns the joined path with the provided directory, or
// if cdback is true, returns the path's directory.
func GetListPath(path, dir string, cdback bool) (string, string) {
	if filepath.Dir(path) == "." && cdback {
		return "", ""
	}

	path = filepath.Clean(path)

	if !cdback {
		path = filepath.Join(path, dir)
	} else {
		path = filepath.Dir(path)
		dir = filepath.Base(path)
	}

	return path, dir
}
