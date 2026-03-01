package test

import (
	"path/filepath"
	"salesmate/cron"
	"testing"
)

// TestCronFunctionality performs functional tests for the cron functionality
func TestCronFunctionality(t *testing.T) {
	// Create a temporary file for the cron store
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "cron_store.json")

	// Create a new cron service
	service, err := cron.NewCronService(storePath)
	if err != nil {
		t.Fatalf("Failed to create cron service: %v", err)
	}

	// Define a simple callback function for jobs
	callback := func(job *cron.CronJob) (string, error) {
		// Simulate job execution
		return "Job " + job.Name + " executed", nil
	}
	service.SetOnJobCallback(callback)

	// Test adding a job with cron schedule
	cronSchedule := cron.CronSchedule{
		Kind: "cron",
		Expr: "*/1 * * * *", // Every minute (just for testing, won't actually run every minute in test)
	}

	job1, err := service.AddJob("test-job", cronSchedule, "Test cron job message", true, "user123", "telegram", false)
	if err != nil {
		t.Errorf("Failed to add cron job: %v", err)
	} else {
		t.Logf("✓ Added cron job with ID: %s", job1.ID)
	}

	// Test getting jobs
	jobs := service.ListJobs(true) // Include disabled
	t.Logf("✓ Retrieved %d cron jobs", len(jobs))

	// Test removing a job
	success := service.RemoveJob(job1.ID)
	if !success {
		t.Errorf("Failed to remove cron job with ID: %s", job1.ID)
	} else {
		t.Logf("✓ Removed cron job with ID: %s", job1.ID)
	}

	// Check that no jobs remain
	remainingJobs := service.ListJobs(true)
	if len(remainingJobs) != 0 {
		t.Errorf("Expected 0 jobs remaining, got %d", len(remainingJobs))
	} else {
		t.Logf("✓ Confirmed that no jobs remain after removal")
	}

	// Add a job without any recurring schedule (just for state testing)
	singleSchedule := cron.CronSchedule{
		Kind: "at",
		AtMS: 0, // Not scheduled to run
	}

	job2, err := service.AddJob("single-job", singleSchedule, "Test single job message", true, "user456", "discord", false)
	if err != nil {
		t.Errorf("Failed to add single job: %v", err)
	} else {
		t.Logf("✓ Added single job with ID: %s", job2.ID)
	}

	// Test enabling/disabling a job
	disabledJob := service.EnableJob(job2.ID, false)
	if disabledJob == nil {
		t.Error("Failed to disable job")
	} else {
		t.Logf("✓ Disabled job with ID: %s", job2.ID)
	}

	// Re-enable the job
	enabledJob := service.EnableJob(job2.ID, true)
	if enabledJob == nil {
		t.Error("Failed to enable job")
	} else {
		t.Logf("✓ Re-enabled job with ID: %s", job2.ID)
	}

	// Test manual job execution
	success = service.RunJob(job2.ID, true) // Force run even if disabled
	if !success {
		t.Errorf("Failed to manually run job: %s", job2.ID)
	} else {
		t.Logf("✓ Manually ran job with ID: %s", job2.ID)
	}

	// Test service status
	status := service.Status()
	if count, ok := status["jobs"].(int); ok {
		t.Logf("✓ Service status shows %d jobs", count)
	} else {
		t.Error("Service status doesn't contain expected job count")
	}

	t.Logf("✓ Cron functionality test completed (without recurring schedules to prevent hanging)")
}

// TestCronInvalidSchedule tests behavior with invalid schedules
func TestCronInvalidSchedule(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "cron_store_invalid.json")

	service, err := cron.NewCronService(storePath)
	if err != nil {
		t.Fatalf("Failed to create cron service: %v", err)
	}

	callback := func(job *cron.CronJob) (string, error) {
		return "Executed", nil
	}
	service.SetOnJobCallback(callback)

	// Test adding a job with invalid cron expression
	invalidSchedule := cron.CronSchedule{
		Kind: "cron",
		Expr: "invalid-expression", // Invalid cron expression
	}

	_, err = service.AddJob("invalid-job", invalidSchedule, "Invalid job message", true, "user123", "telegram", false)
	if err == nil {
		t.Error("Expected error when adding job with invalid schedule")
	} else {
		t.Logf("✓ Properly rejected invalid cron schedule: %v", err)
	}

	// Test adding a job with unsupported schedule type
	unsupportedSchedule := cron.CronSchedule{
		Kind: "unsupported_type",
		Expr: "* * * * *",
	}

	_, err = service.AddJob("unsupported-job", unsupportedSchedule, "Unsupported job message", true, "user123", "telegram", false)
	if err != nil {
		t.Logf("✓ Handled unsupported schedule type: %v", err)
	}

	// Test removing a non-existent job
	success := service.RemoveJob("non-existent-job")
	if success {
		t.Error("Should not return success when removing non-existent job")
	} else {
		t.Logf("✓ Properly handled removal of non-existent job")
	}

	t.Logf("✓ Cron invalid schedule test completed")
}
