package rclone

import (
	"context"

	"github.com/darkhz/rclone-tui/rclone"
)

// About stores the storage information for a remote.
type About struct {
	Total   int64 `json:"total"`
	Used    int64 `json:"used"`
	Trashed int64 `json:"trashed"`
	Other   int64 `json:"other"`
	Free    int64 `json:"free"`
}

// AboutFS returns the storage information for a remote.
func AboutFS(ctx context.Context, fs string) (About, error) {
	var about About

	command := map[string]interface{}{
		"fs": fs,
	}

	response, err := rclone.SendCommand(command, "/operations/about", ctx)
	if err != nil {
		return About{}, err
	}

	err = response.Decode(&about)
	if err != nil {
		return About{}, err
	}

	return about, nil
}
