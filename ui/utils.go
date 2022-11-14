package ui

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	rcfns "github.com/darkhz/rclone-tui/rclone/operations"
)

// MatchProvider returns true if provider matches the providerConfig string.
// Taken from: https://github.com/rclone/rclone/blob/master/fs/backend_config.go#L542
func MatchProvider(providerConfig, provider string) bool {
	if providerConfig == "" || provider == "" {
		return true
	}
	negate := false
	if strings.HasPrefix(providerConfig, "!") {
		providerConfig = providerConfig[1:]
		negate = true
	}
	providers := strings.Split(providerConfig, ",")
	matched := false
	for _, p := range providers {
		if p == provider {
			matched = true
			break
		}
	}
	if negate {
		return !matched
	}
	return matched
}

// ReadableString parses d into a human-readable duration.
// Taken from: https://github.com/rclone/rclone/blob/master/fs/parseduration.go#L132
func ReadableString(d time.Duration) string {
	switch d {
	case 0:
		return "0s"
	}

	readableString := ""

	// Check for minus durations.
	if d < 0 {
		readableString += "-"
	}

	duration := time.Duration(math.Abs(float64(d)))

	// Convert duration.
	seconds := int64(duration.Seconds()) % 60
	minutes := int64(duration.Minutes()) % 60
	hours := int64(duration.Hours()) % 24
	days := int64(duration/(24*time.Hour)) % 365 % 7

	// Edge case between 364 and 365 days.
	// We need to calculate weeks from what is left from years
	leftYearDays := int64(duration/(24*time.Hour)) % 365
	weeks := leftYearDays / 7
	if leftYearDays >= 364 && leftYearDays < 365 {
		weeks = 52
	}

	years := int64(duration/(24*time.Hour)) / 365
	milliseconds := int64(duration/time.Millisecond) -
		(seconds * 1000) - (minutes * 60000) - (hours * 3600000) -
		(days * 86400000) - (weeks * 604800000) - (years * 31536000000)

	// Create a map of the converted duration time.
	durationMap := map[string]int64{
		"ms": milliseconds,
		"s":  seconds,
		"m":  minutes,
		"h":  hours,
		"d":  days,
		"w":  weeks,
		"y":  years,
	}

	// Construct duration string.
	for _, u := range [...]string{"y", "w", "d", "h", "m", "s", "ms"} {
		v := durationMap[u]
		strval := strconv.FormatInt(v, 10)
		if v == 0 {
			continue
		}
		readableString += strval + u + " "
	}

	return readableString
}

// Normalize returns a set of numbers on the interval [0,1] for a given set of inputs.
// Adapted from: https://github.com/blend/go-sdk/blob/master/mathutil/normalize.go#L13
func Normalize(values ...float64) []float64 {
	var max float64

	if len(values) == 0 {
		return values
	}

	for _, v := range values {
		if v > max {
			max = v
		}
	}

	output := make([]float64, len(values))
	for x, v := range values {
		output[x] = RoundDown(v/max, 0.0001)
	}
	return output
}

// RoundDown rounds down to a given roundTo value.
// Taken from: https://github.com/blend/go-sdk/blob/master/mathutil/round.go#L18
func RoundDown(value, roundTo float64) float64 {
	d1 := math.Floor(value / roundTo)
	return d1 * roundTo
}

// parseDataMap parses the user-input form data into an rclone parseable format.
func parseDataMap(origData map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})

	for key, value := range origData {
		if submap, ok := value.(map[string]interface{}); ok {
			data[key] = parseDataMap(submap)
			continue
		}

		if b, ok := value.(bool); ok {
			data[key] = b
			continue
		}

		v, ok := value.(string)
		if !ok {
			continue
		}

		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			data[key] = i
		} else if f, err := strconv.ParseFloat(v, 64); err == nil {
			data[key] = f
		} else {
			data[key] = value
		}
	}

	return data
}

// sortList sorts a list of directory entries according to the order and mode.
func sortList(listItems []rcfns.ListItem, asc bool, mode string) {
	sort.Slice(listItems, func(i, j int) bool {
		var a, b int

		if listItems[i].IsDir != listItems[j].IsDir {
			return listItems[i].IsDir
		}

		if asc {
			a, b = i, j
		} else {
			a, b = j, i
		}

		switch mode {
		case "size":
			return listItems[a].Size < listItems[b].Size

		case "modified":
			return listItems[a].ModifiedTimeUnix < listItems[b].ModifiedTimeUnix
		}

		return listItems[a].Name < listItems[b].Name
	})
}

// modifyDataMap either returns the string representation of the data if set it false, or
// it sets the data within the map if set is true.
func modifyDataMap(datamap map[string]interface{}, key string, value interface{}, set bool) string {
	if set {
		if datamap == nil {
			datamap = make(map[string]interface{})
		}

		datamap[key] = value

		return ""
	}

	if datamap == nil {
		return ""
	}

	switch data := datamap[key].(type) {
	case bool:
		return strconv.FormatBool(data)

	case string:
		return data
	}

	return ""
}
