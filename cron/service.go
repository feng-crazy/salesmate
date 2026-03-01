package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronPayload represents the payload for a cron job
type CronPayload struct {
	Message string `json:"message"`
	Channel string `json:"channel,omitempty"`
	To      string `json:"to,omitempty"`
	Deliver bool   `json:"deliver"`
}

// CronSchedule represents the schedule for a job
type CronSchedule struct {
	Kind    string `json:"kind"`      // "every", "cron", "at"
	EveryMS *int64 `json:"every_ms,omitempty"`
	Expr    string `json:"expr,omitempty"`
	AtMS    int64  `json:"at_ms,omitempty"`
	Tz      string `json:"tz,omitempty"`
}

// CronState represents the state of a job
type CronState struct {
	NextRunAtMS *int64 `json:"next_run_at_ms,omitempty"`
}

// CronJob represents a scheduled job
type CronJob struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Schedule CronSchedule  `json:"schedule"`
	Payload  CronPayload   `json:"payload"`
	State    CronState     `json:"state"`
	Enabled  bool          `json:"enabled"`
	DeleteAfterRun bool   `json:"delete_after_run,omitempty"`
}

// CronService manages scheduled jobs
type CronService struct {
	storePath string
	jobs      map[string]*CronJob
	cron      *cron.Cron
	mutex     sync.RWMutex
	onJob     func(job *CronJob) (string, error)
}

// NewCronService creates a new cron service
func NewCronService(storePath string) (*CronService, error) {
	dir := filepath.Dir(storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	service := &CronService{
		storePath: storePath,
		jobs:      make(map[string]*CronJob),
		cron:      cron.New(),
	}

	// Load existing jobs
	if err := service.loadJobs(); err != nil {
		return nil, fmt.Errorf("failed to load jobs: %w", err)
	}

	return service, nil
}

// loadJobs loads jobs from the store file
func (cs *CronService) loadJobs() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	data, err := os.ReadFile(cs.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}

	var jobs []*CronJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return fmt.Errorf("failed to unmarshal jobs: %w", err)
	}

	for _, job := range jobs {
		cs.jobs[job.ID] = job
		if job.Enabled {
			cs.scheduleJob(job)
		}
	}

	return nil
}

// saveJobs saves jobs to the store file
func (cs *CronService) saveJobs() error {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	var jobs []*CronJob
	for _, job := range cs.jobs {
		jobs = append(jobs, job)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(cs.storePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// AddJob adds a new scheduled job
func (cs *CronService) AddJob(name string, schedule CronSchedule, message string, deliver bool, to string, channel string, deleteAfterRun bool) (*CronJob, error) {
	job := &CronJob{
		ID:               fmt.Sprintf("job_%d", time.Now().Unix()),
		Name:             name,
		Schedule:         schedule,
		Payload:          CronPayload{Message: message, Deliver: deliver, To: to, Channel: channel},
		State:            CronState{},
		Enabled:          true,
		DeleteAfterRun:   deleteAfterRun,
	}

	cs.mutex.Lock()
	cs.jobs[job.ID] = job
	cs.scheduleJob(job)
	cs.mutex.Unlock()

	if err := cs.saveJobs(); err != nil {
		return nil, err
	}

	return job, nil
}

// RemoveJob removes a job by ID
func (cs *CronService) RemoveJob(jobID string) bool {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	job, exists := cs.jobs[jobID]
	if !exists {
		return false
	}

	delete(cs.jobs, jobID)
	if job.Enabled {
		// Note: We can't easily unschedule jobs in the cron lib without storing entry IDs
		// This is a limitation of the library; for now we'll restart the cron scheduler
		newCron := cron.New()
		for _, existingJob := range cs.jobs {
			if existingJob.Enabled {
				cs.scheduleJob(existingJob)
			}
		}
		cs.cron.Stop()
		cs.cron = newCron
	}

	if err := cs.saveJobs(); err != nil {
		return false // revert deletion?
	}

	return true
}

// ListJobs returns all scheduled jobs
func (cs *CronService) ListJobs(includeDisabled bool) []*CronJob {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	var jobs []*CronJob
	for _, job := range cs.jobs {
		if includeDisabled || job.Enabled {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

// EnableJob enables or disables a job
func (cs *CronService) EnableJob(jobID string, enabled bool) *CronJob {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	job, exists := cs.jobs[jobID]
	if !exists {
		return nil
	}

	job.Enabled = enabled
	if enabled {
		cs.scheduleJob(job)
	} else {
		// For now, we restart the scheduler to remove the job
		newCron := cron.New()
		for _, existingJob := range cs.jobs {
			if existingJob.Enabled && existingJob.ID != jobID {
				cs.scheduleJob(existingJob)
			}
		}
		cs.cron.Stop()
		cs.cron = newCron
	}

	if err := cs.saveJobs(); err != nil {
		return nil
	}

	return job
}

// RunJob manually runs a job
func (cs *CronService) RunJob(jobID string, force bool) bool {
	cs.mutex.RLock()
	job, exists := cs.jobs[jobID]
	cs.mutex.RUnlock()

	if !exists || (!job.Enabled && !force) {
		return false
	}

	go func() {
		if cs.onJob != nil {
			_, err := cs.onJob(job)
			if err != nil {
				// Log error
				fmt.Printf("Error running job %s: %v\n", jobID, err)
			}

			// Delete job if it's one-time and marked for deletion
			if job.DeleteAfterRun {
				cs.RemoveJob(jobID)
			}
		}
	}()

	return true
}

// Status returns service status
func (cs *CronService) Status() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return map[string]interface{}{
		"jobs": len(cs.jobs),
	}
}

// SetOnJobCallback sets the callback function to execute jobs
func (cs *CronService) SetOnJobCallback(callback func(job *CronJob) (string, error)) {
	cs.onJob = callback
}

// scheduleJob schedules a job based on its schedule type
func (cs *CronService) scheduleJob(job *CronJob) {
	if !job.Enabled {
		return
	}

	switch job.Schedule.Kind {
	case "every":
		if job.Schedule.EveryMS != nil {
			duration := time.Duration(*job.Schedule.EveryMS) * time.Millisecond
			go func() {
				ticker := time.NewTicker(duration)
				defer ticker.Stop()

				for range ticker.C {
					if !job.Enabled {
						break
					}

					if cs.onJob != nil {
						_, err := cs.onJob(job)
						if err != nil {
							fmt.Printf("Error running job %s: %v\n", job.ID, err)
						}

						// Delete job if it's one-time and marked for deletion
						if job.DeleteAfterRun {
							cs.RemoveJob(job.ID)
							break
						}
					}
				}
			}()
		}
	case "cron":
		if job.Schedule.Expr != "" {
			var err error

			if job.Schedule.Tz != "" {
				// Parse with timezone
				_, locErr := time.LoadLocation(job.Schedule.Tz)
				if locErr != nil {
					fmt.Printf("Invalid timezone '%s' for job %s: %v\n", job.Schedule.Tz, job.ID, locErr)
					return
				}

				// Use the standard cron parser with timezone
				parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
				_, err := parser.Parse(job.Schedule.Expr)
				if err != nil {
					fmt.Printf("Invalid cron expression '%s' for job %s: %v\n", job.Schedule.Expr, job.ID, err)
					return
				}

				// For timezone support, we'll run the cron with UTC and handle the timezone ourselves
				// This is a simplified implementation - in a real application you'd need proper timezone handling
				_, err = cs.cron.AddFunc(job.Schedule.Expr, func() {
					if cs.onJob != nil {
						_, runErr := cs.onJob(job)
						if runErr != nil {
							fmt.Printf("Error running job %s: %v\n", job.ID, runErr)
						}

						// Delete job if it's one-time and marked for deletion
						if job.DeleteAfterRun {
							cs.RemoveJob(job.ID)
						}
					}
				})
			} else {
				_, err = cs.cron.AddFunc(job.Schedule.Expr, func() {
					if cs.onJob != nil {
						_, runErr := cs.onJob(job)
						if runErr != nil {
							fmt.Printf("Error running job %s: %v\n", job.ID, runErr)
						}

						// Delete job if it's one-time and marked for deletion
						if job.DeleteAfterRun {
							cs.RemoveJob(job.ID)
						}
					}
				})
			}

			if err != nil {
				fmt.Printf("Failed to schedule job %s: %v\n", job.ID, err)
				return
			}
		}
	case "at":
		// One-time execution at specific time
		go func() {
			atTime := time.UnixMilli(job.Schedule.AtMS)
			now := time.Now()

			if now.After(atTime) {
				// Time has passed, run immediately if DeleteAfterRun is true
				if cs.onJob != nil && job.DeleteAfterRun {
					_, err := cs.onJob(job)
					if err != nil {
						fmt.Printf("Error running job %s: %v\n", job.ID, err)
					}
					cs.RemoveJob(job.ID)
				}
				return
			}

			time.Sleep(atTime.Sub(now))

			if cs.onJob != nil {
				_, err := cs.onJob(job)
				if err != nil {
					fmt.Printf("Error running job %s: %v\n", job.ID, err)
				}

				if job.DeleteAfterRun {
					cs.RemoveJob(job.ID)
				}
			}
		}()
	}
}

// jobFunc implements cron.Job interface
type jobFunc struct {
	job     *CronJob
	service *CronService
}

func (jf *jobFunc) Run() {
	if jf.service.onJob != nil {
		_, err := jf.service.onJob(jf.job)
		if err != nil {
			fmt.Printf("Error running job %s: %v\n", jf.job.ID, err)
		}

		// Delete job if it's one-time and marked for deletion
		if jf.job.DeleteAfterRun {
			jf.service.RemoveJob(jf.job.ID)
		}
	}
}

// Start starts the cron service
func (cs *CronService) Start() {
	cs.cron.Start()
}

// Stop stops the cron service
func (cs *CronService) Stop() {
	cs.cron.Stop()
}