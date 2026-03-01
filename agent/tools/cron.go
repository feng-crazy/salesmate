package tools

import (
	"fmt"
	"salesmate/cron"
)

// CronTool implements a tool for scheduling reminders and tasks
type CronTool struct {
	cronService *cron.CronService
	channel     string
	chatID      string
}

// NewCronTool creates a new cron tool
func NewCronTool(cronService *cron.CronService) *CronTool {
	return &CronTool{
		cronService: cronService,
	}
}

// SetContext sets the current session context for delivery
func (t *CronTool) SetContext(channel string, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

// Name returns the name of the tool
func (t *CronTool) Name() string {
	return "cron"
}

// Description returns the description of the tool
func (t *CronTool) Description() string {
	return "Schedule reminders and recurring tasks. Actions: add, list, remove."
}

// Call executes the tool with the given arguments
func (t *CronTool) Call(args map[string]interface{}) (string, error) {
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' argument")
	}

	switch action {
	case "add":
		return t.addJob(args)
	case "list":
		return t.listJobs()
	case "remove":
		return t.removeJob(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// addJob adds a new scheduled job
func (t *CronTool) addJob(args map[string]interface{}) (string, error) {
	message, ok := args["message"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'message' argument for add action")
	}

	if t.channel == "" || t.chatID == "" {
		return "", fmt.Errorf("no session context (channel/chat_id)")
	}

	// Parse schedule options
	var schedule cron.CronSchedule
	var err error

	if everySeconds, ok := args["every_seconds"].(float64); ok {
		everyMS := int64(everySeconds) * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else if cronExpr, ok := args["cron_expr"].(string); ok {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
		if tz, ok := args["tz"].(string); ok {
			schedule.Tz = tz
		}
	} else if atValue, ok := args["at"].(string); ok {
		// Parse ISO time format
		// This is a simplified version - would need proper time parsing
		// For now, we'll just return a message indicating it's not implemented
		_ = atValue // Use the variable to avoid "not used" error
		return "", fmt.Errorf("'at' scheduling not fully implemented yet in this version")
	} else {
		return "", fmt.Errorf("either 'every_seconds', 'cron_expr', or 'at' is required for add action")
	}

	// Check timezone validity if provided
	if schedule.Tz != "" && schedule.Kind == "cron" {
		// In a real implementation, we would validate the timezone
		// For now, we'll just accept it
	}

	// Add the job to cron service
	job, err := t.cronService.AddJob(message, schedule, message, true, t.chatID, t.channel, schedule.Kind == "at")
	if err != nil {
		return "", fmt.Errorf("failed to add job: %w", err)
	}

	return fmt.Sprintf("Created job '%s' (id: %s)", job.Name, job.ID), nil
}

// listJobs lists all scheduled jobs
func (t *CronTool) listJobs() (string, error) {
	jobs := t.cronService.ListJobs(false) // Don't include disabled jobs

	if len(jobs) == 0 {
		return "No scheduled jobs.", nil
	}

	result := "Scheduled jobs:\n"
	for _, job := range jobs {
		result += fmt.Sprintf("- %s (id: %s, %s)\n", job.Name, job.ID, job.Schedule.Kind)

		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			result += fmt.Sprintf("  Every %d seconds\n", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			result += fmt.Sprintf("  Cron: %s", job.Schedule.Expr)
			if job.Schedule.Tz != "" {
				result += fmt.Sprintf(" (timezone: %s)", job.Schedule.Tz)
			}
			result += "\n"
		} else if job.Schedule.Kind == "at" {
			result += fmt.Sprintf("  At: %d (timestamp)\n", job.Schedule.AtMS)
		}
	}

	return result, nil
}

// removeJob removes a scheduled job
func (t *CronTool) removeJob(args map[string]interface{}) (string, error) {
	jobID, ok := args["job_id"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'job_id' argument for remove action")
	}

	if t.cronService.RemoveJob(jobID) {
		return fmt.Sprintf("Removed job %s", jobID), nil
	}

	return fmt.Sprintf("Job %s not found", jobID), nil
}
