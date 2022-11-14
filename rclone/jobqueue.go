package rclone

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Job stores the rclone job information.
type Job struct {
	ID      int64
	Group   string
	Context context.Context

	Type        string
	Description string
	Updates     chan JobInfo
	Cancel      context.CancelFunc

	RefreshItems interface{}
}

// JobInfo stores the rclone running job stats.
type JobInfo struct {
	Duration  float64                `json:"duration"`
	EndTime   time.Time              `json:"endTime"`
	Error     string                 `json:"error"`
	Finished  bool                   `json:"finished"`
	Group     string                 `json:"group"`
	ID        int64                  `json:"id"`
	Output    map[string]interface{} `json:"output"`
	StartTime time.Time              `json:"startTime"`
	Success   bool                   `json:"success"`
	Transfers struct {
		Stats []TransferStat `json:"transferring"`
	}

	JobCount          int64
	Type, Description string
	CurrentTransfer   TransferStat

	RefreshItems interface{}
}

// TransferStat stores the file transfer stats.
type TransferStat struct {
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Bytes      int64   `json:"bytes,omitempty"`
	Eta        int64   `json:"eta,omitempty"`
	Group      string  `json:"group,omitempty"`
	Percentage int64   `json:"percentage,omitempty"`
	Speed      float64 `json:"speed,omitempty"`
	SpeedAvg   float64 `json:"speedAvg,omitempty"`
}

var (
	jobQueue sync.Map

	jobLock     sync.Mutex
	jobTotal    int64
	JobInfoChan chan JobInfo
)

// NewJob returns a job with the provided type, description, and optional group.
func NewJob(jobType, jobDesc string, jobId int64, jobGroup ...string) *Job {
	var group string

	if jobGroup != nil {
		group = jobGroup[0]
	}

	jobChan := make(chan JobInfo, 10)
	context, cancel := context.WithCancel(context.Background())

	job := Job{
		ID:      jobId,
		Group:   group,
		Context: context,

		Type:        jobType,
		Updates:     jobChan,
		Description: jobDesc,
		Cancel:      cancel,
	}

	return &job
}

// AddJobToQueue adds a job to the queue and if nomonitor is not set, will automatically
// start to monitor the job.
func AddJobToQueue(job *Job, nomonitor ...struct{}) *Job {
	jobMap, ok := jobQueue.Load(job.Type)
	if !ok {
		jobMap = make(map[int64]*Job)
	}

	jobMap.(map[int64]*Job)[job.ID] = job

	jobQueue.Store(job.Type, jobMap)

	modifyJobCount(job.Type, true)

	if nomonitor == nil {
		go MonitorJob(job)
	}

	return job
}

// GetJobReply polls the job status and returns its output information
// when the job has finished.
func GetJobReply(job *Job) (JobInfo, error) {
	var info JobInfo
	var err error

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

MonitorStatus:
	for {
		select {
		case info = <-job.Updates:
			switch {
			case info.Error != "":
				err = fmt.Errorf(info.Error)
				fallthrough

			case info.Finished:
				break MonitorStatus
			}

		case <-t.C:
		}
	}

	return info, err
}

// GetJobQueue returns the job queue.
func GetJobQueue() *sync.Map {
	return &jobQueue
}

// GetNewJobID generates a new job ID.
func GetNewJobID(jobType string) int64 {
	_, jobId := getJobTypeInfo(jobType)
	if jobId == -1 {
		return 0
	}

	return jobId + 1
}

// GetLatestJob gets the latest job associated with the provided job type.
func GetLatestJob(jobType string) (*Job, error) {
	err := fmt.Errorf("Cannot get job information for " + jobType)

	jobMap, jobId := getJobTypeInfo(jobType)
	if jobMap == nil {
		return nil, err
	}

	return jobMap[jobId], nil
}

// JobInfoStatus returns the job stats update channel.
func JobInfoStatus() chan JobInfo {
	jobLock.Lock()
	defer jobLock.Unlock()

	if JobInfoChan == nil {
		JobInfoChan = make(chan JobInfo, 100)
	}

	return JobInfoChan
}

// MonitorJob monitors the provided job, and if nostop is not set, it will
// automatically stop monitoring the job.
//
//gocyclo:ignore
func MonitorJob(job *Job, nostop ...struct{}) {
	var jobInfo JobInfo

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	if job.Group == "" {
		job.Group = "job/" + strconv.FormatInt(job.ID, 10)
	}

	command := map[string]interface{}{
		"jobid": job.ID,
		"group": job.Group,
	}

	for {
		for _, endpoint := range []string{
			"/job/status",
			"/core/stats",
		} {
			response, err := SendCommand(command, endpoint)
			if err != nil {
				jobInfo.Error = err.Error()
				goto SendInfo
			}

			if strings.Contains(endpoint, "job") {
				err = response.Decode(&jobInfo)
				if err != nil {
					jobInfo.Error = err.Error()
					goto SendInfo
				}

				continue
			}

			err = response.Decode(&jobInfo.Transfers)
			if err != nil {
				jobInfo.Error = err.Error()
			}
		}

		jobInfo.Type = job.Type
		jobInfo.JobCount = jobCount()
		jobInfo.Description = job.Description

		for _, transfer := range jobInfo.Transfers.Stats {
			if transfer.Group == job.Group {
				jobInfo.CurrentTransfer = transfer
				break
			}
		}

		jobInfo.Transfers.Stats = nil

	SendInfo:
		select {
		case job.Updates <- jobInfo:
			select {
			case JobInfoStatus() <- jobInfo:

			default:
			}

			if jobInfo.Error != "" || jobInfo.Finished == true {
				if nostop == nil {
					StopJob(job, jobInfo.Error)
				}

				return
			}

		default:
		}

		select {
		case <-t.C:

		case <-job.Context.Done():
			jobInfo.Error = job.Description + " cancelled"
			goto SendInfo
		}
	}
}

// StopJob stops the provided job.
func StopJob(job *Job, errors string, force ...struct{}) {
	command := map[string]interface{}{
		"jobid": job.ID,
	}

	if force != nil {
		jobQueue.Delete(job.Type)
		goto JobFinished
	}

	SendCommandAsync(job.Type, job.Description, command, "/job/stop", struct{}{})

	if jobMap, ok := jobQueue.Load(job.Type); ok {
		delete(jobMap.(map[int64]*Job), job.ID)

		if len(jobMap.(map[int64]*Job)) == 0 {
			jobQueue.Delete(job.Type)
		} else {
			jobQueue.Store(job.Type, jobMap)
		}
	}

JobFinished:
	jobFinished := JobInfo{
		ID:          job.ID,
		Type:        job.Type,
		Description: job.Description,
		JobCount:    modifyJobCount(job.Type, false),

		Error:    errors,
		Finished: true,

		RefreshItems: job.RefreshItems,
	}

	select {
	case JobInfoStatus() <- jobFinished:

	default:
	}
}

// StopJobGroup stops all jobs associated with the group.
func StopJobGroup(job *Job) {
	GetJobQueue().Range(func(key, value any) bool {
		jobType := key.(string)
		if jobType != job.Type {
			return true
		}

		jobMap := value.(map[int64]*Job)

		for _, j := range jobMap {
			j.Cancel()
		}

		return false
	})
}

// getJobTypeInfo returns the information and an ID for the provided job type.
func getJobTypeInfo(jobType string) (map[int64]*Job, int64) {
	var jobIds []int64

	jobMap, ok := jobQueue.Load(jobType)
	if !ok {
		return nil, -1
	}

	for id := range jobMap.(map[int64]*Job) {
		jobIds = append(jobIds, id)
	}
	sort.Slice(jobIds, func(i, j int) bool {
		return jobIds[i] < jobIds[j]
	})

	if len(jobIds) == 0 {
		return nil, -1
	}

	return jobMap.(map[int64]*Job), jobIds[len(jobIds)-1]
}

// modifyJobCount modified the total job count.
func modifyJobCount(jobType string, inc bool) int64 {
	jobLock.Lock()
	defer jobLock.Unlock()

	if strings.HasPrefix(jobType, "UI:") ||
		strings.HasPrefix(jobType, "_") {
		return -1
	}

	if inc {
		jobTotal++
	} else {
		jobTotal--
	}

	return jobTotal
}

// jobCount returns the job count.
func jobCount() int64 {
	jobLock.Lock()
	defer jobLock.Unlock()

	return jobTotal
}
