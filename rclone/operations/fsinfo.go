package rclone

import (
	"sort"

	"github.com/darkhz/rclone-tui/rclone"
	"github.com/mitchellh/mapstructure"
)

// FsDetail stores details of a remote.
type FsDetail struct {
	Name      string          `json:"Name"`
	Precision int             `json:"Precision"`
	Root      string          `json:"Root"`
	String    string          `json:"String"`
	Hashes    []string        `json:"Hashes"`
	Features  map[string]bool `json:"Features"`

	FeatureList []string
}

// FsInfo returns information about a remote.
func FsInfo(id, fs string) (FsDetail, error) {
	var detail FsDetail

	command := map[string]interface{}{
		"fs": fs,
	}

	job, err := rclone.SendCommandAsync("UI:Explorer:"+id, "Getting fs information", command, "/operations/fsinfo")
	if err != nil {
		return FsDetail{}, err
	}

	jobInfo, err := rclone.GetJobReply(job)
	if err != nil {
		return FsDetail{}, err
	}

	err = mapstructure.Decode(jobInfo.Output, &detail)
	if err != nil {
		return FsDetail{}, err
	}

	for feature, exist := range detail.Features {
		if exist {
			detail.FeatureList = append(detail.FeatureList, feature)
		}
	}

	detail.Features = nil
	sort.Slice(detail.FeatureList, func(i, j int) bool {
		return detail.FeatureList[i] < detail.FeatureList[j]
	})

	return detail, nil
}
