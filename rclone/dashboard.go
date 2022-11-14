package rclone

import (
	"sync"
	"time"
)

// DashboardInfo stores the dashboard information.
type DashboardInfo struct {
	Connected bool
	Bandwidth string
	Stats     *DashboardStats

	User    string
	Version string
}

// DashboardStats stores the rclone stats.
type DashboardStats struct {
	Bytes          int64          `json:"bytes"`
	Checks         int64          `json:"checks"`
	DeletedDirs    int64          `json:"deletedDirs"`
	Deletes        int64          `json:"deletes"`
	ElapsedTime    float64        `json:"elapsedTime"`
	Errors         int64          `json:"errors"`
	Eta            int64          `json:"eta"`
	FatalError     bool           `json:"fatalError"`
	Renames        int64          `json:"renames"`
	RetryError     bool           `json:"retryError"`
	Speed          float64        `json:"speed"`
	TotalBytes     int64          `json:"totalBytes"`
	TotalChecks    int64          `json:"totalChecks"`
	TotalTransfers int64          `json:"totalTransfers"`
	TransferTime   int64          `json:"transferTime"`
	Transferring   []TransferStat `json:"transferring"`
	Transfers      int64          `json:"transfers"`
}

var (
	dashExit                 chan struct{}
	dashCheck, dashConnected chan bool

	dashLock sync.Mutex
)

// StartDashboard starts polling for rclone stats.
func StartDashboard() (chan DashboardInfo, chan struct{}) {
	dashInfo := make(chan DashboardInfo)
	exit := make(chan struct{}, 1)

	dashExit = exit

	go updateDashboard(dashInfo, exit)

	return dashInfo, exit
}

// StopDashboard stops polling for rclone stats.
func StopDashboard() {
	if dashExit == nil {
		return
	}

	select {
	case dashExit <- struct{}{}:

	default:
	}
}

// PollConnection polls the connectivity status of the host.
func PollConnection(updateMonitor bool) chan bool {
	dashLock.Lock()
	defer dashLock.Unlock()

	if dashConnected != nil && dashCheck != nil {
		goto ConnectionChannel
	}

	dashCheck = make(chan bool)
	dashConnected = make(chan bool)

	go func() {
		for {
			connected := DialServer() == nil

			select {
			case dashConnected <- connected:

			default:
			}

			select {
			case dashCheck <- connected:

			default:
			}

			time.Sleep(1 * time.Second)
		}
	}()

ConnectionChannel:
	if updateMonitor {
		return dashCheck
	}

	return dashConnected
}

// updateDashboard updates the rclone stats.
func updateDashboard(dashInfo chan DashboardInfo, exit chan struct{}) {
	var connected bool

	data := []interface{}{
		nil,
		new(DashboardStats),
		map[string]interface{}{},
	}

	for {
		select {
		case <-exit:
			return

		case connected = <-PollConnection(true):
		}

		if connected {
			version, err := GetVersion(false)
			if err == nil {
				data[0] = version.Version + " (" + version.Arch + ")"
			}

			for i, endpoint := range []string{
				"/core/stats",
				"/core/bwlimit",
			} {
				response, err := SendCommand(map[string]interface{}{}, endpoint)
				if err != nil {
					continue
				}

				err = response.Decode(&data[i+1])
				if err != nil {
					continue
				}
			}
		}

		info := DashboardInfo{
			Connected: connected,
		}

		if version, ok := data[0].(string); ok {
			info.Version = version
		}
		if stats, ok := data[1].(*DashboardStats); ok {
			info.Stats = stats
		}
		if bandwidth, ok := data[2].(map[string]interface{}); ok {
			if bw, ok := bandwidth["rate"].(string); ok {
				info.Bandwidth = bw
			}
		}

		select {
		case dashInfo <- info:

		default:
		}
	}
}
