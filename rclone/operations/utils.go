package rclone

import (
	"context"

	"github.com/darkhz/rclone-tui/rclone"
)

// GetDataSlice runs a command and returns its output as a slice.
func GetDataSlice(ctx context.Context, endpoint, key string) ([]string, error) {
	var data []string
	var dataMap map[string]interface{}

	res, err := rclone.SendCommand(map[string]interface{}{}, endpoint, ctx)
	if err != nil {
		return nil, err
	}

	err = res.Decode(&dataMap)
	if err != nil {
		return nil, err
	}

	for _, remote := range dataMap[key].([]interface{}) {
		data = append(data, remote.(string))
	}

	return data, nil
}
