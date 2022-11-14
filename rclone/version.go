package rclone

type Version struct {
	Arch      string `json:"arch"`
	GoTags    string `json:"goTags"`
	GoVersion string `json:"goVersion"`
	IsBeta    bool   `json:"isBeta"`
	IsGit     bool   `json:"isGit"`
	Linking   string `json:"linking"`
	Os        string `json:"os"`
	Version   string `json:"version"`
}

var version Version

// GetVersion returns the version.
func GetVersion(force bool) (Version, error) {
	if version != (Version{}) && !force {
		return version, nil
	}

	res, err := SendCommand(map[string]interface{}{}, "/core/version")
	if err != nil {
		return Version{}, err
	}

	err = res.Decode(&version)

	return version, err
}
