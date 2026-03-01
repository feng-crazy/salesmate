package commands

import (
	"fmt"
	"os"
	"time"

	"salesmate/config"
	"salesmate/cron"

	"github.com/spf13/cobra"
)

// cronCmd represents the cron command
var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled tasks",
	Long:  `Manage scheduled tasks and reminders.`,
}

// cronListCmd represents the cron list command
var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled jobs",
	Long:  `List all scheduled jobs.`,
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")

		// Load config to get store path
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dataDir := cfg.GetWorkspacePath() // Use workspace path for simplicity
		storePath := fmt.Sprintf("%s/data/cron/jobs.json", dataDir)

		// Create cron service
		service, err := cron.NewCronService(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cron service: %v\n", err)
			os.Exit(1)
		}

		jobs := service.ListJobs(all)
		if len(jobs) == 0 {
			fmt.Println("No scheduled jobs.")
			return
		}

		fmt.Println("Scheduled jobs:")
		for _, job := range jobs {
			status := "enabled"
			if !job.Enabled {
				status = "disabled"
			}

			fmt.Printf("- %s (id: %s, %s, status: %s)\n", job.Name, job.ID, job.Schedule.Kind, status)
			if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
				everySeconds := *job.Schedule.EveryMS / 1000
				fmt.Printf("  Every: %d seconds\n", everySeconds)
			} else if job.Schedule.Kind == "cron" {
				fmt.Printf("  Expression: %s\n", job.Schedule.Expr)
				if job.Schedule.Tz != "" {
					fmt.Printf("  Timezone: %s\n", job.Schedule.Tz)
				}
			} else if job.Schedule.Kind == "at" {
				atTime := time.UnixMilli(job.Schedule.AtMS)
				fmt.Printf("  At: %s\n", atTime.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  Message: %s\n", job.Payload.Message)
			fmt.Println()
		}
	},
}

// cronAddCmd represents the cron add command
var cronAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a scheduled job",
	Long:  `Add a new scheduled job.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		message, _ := cmd.Flags().GetString("message")
		every, _ := cmd.Flags().GetInt("every")
		cronExpr, _ := cmd.Flags().GetString("cron")
		tz, _ := cmd.Flags().GetString("tz")
		at, _ := cmd.Flags().GetString("at")
		deliver, _ := cmd.Flags().GetBool("deliver")
		to, _ := cmd.Flags().GetString("to")
		channel, _ := cmd.Flags().GetString("channel")

		if tz != "" && cronExpr == "" {
			fmt.Fprintf(os.Stderr, "Error: --tz can only be used with --cron\n")
			os.Exit(1)
		}

		// Create schedule based on inputs
		var schedule cron.CronSchedule
		if every > 0 {
			everyMS := int64(every * 1000)
			schedule = cron.CronSchedule{
				Kind:    "every",
				EveryMS: &everyMS,
			}
		} else if cronExpr != "" {
			schedule = cron.CronSchedule{
				Kind: "cron",
				Expr: cronExpr,
				Tz:   tz,
			}
		} else if at != "" {
			// Parse ISO time format
			dt, err := time.Parse(time.RFC3339, at)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing time: %v\n", err)
				os.Exit(1)
			}
			atMS := dt.UnixMilli()
			schedule = cron.CronSchedule{
				Kind: "at",
				AtMS: atMS,
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: Must specify --every, --cron, or --at\n")
			os.Exit(1)
		}

		// Load config to get store path
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dataDir := cfg.GetWorkspacePath() // Use workspace path for simplicity
		storePath := fmt.Sprintf("%s/data/cron/jobs.json", dataDir)

		// Create cron service
		service, err := cron.NewCronService(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cron service: %v\n", err)
			os.Exit(1)
		}

		// Add job
		job, err := service.AddJob(name, schedule, message, deliver, to, channel, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding job: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added job '%s' (id: %s)\n", job.Name, job.ID)
	},
}

// cronRemoveCmd represents the cron remove command
var cronRemoveCmd = &cobra.Command{
	Use:   "remove [job-id]",
	Short: "Remove a scheduled job",
	Long:  `Remove a scheduled job by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]

		// Load config to get store path
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dataDir := cfg.GetWorkspacePath() // Use workspace path for simplicity
		storePath := fmt.Sprintf("%s/data/cron/jobs.json", dataDir)

		// Create cron service
		service, err := cron.NewCronService(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cron service: %v\n", err)
			os.Exit(1)
		}

		if service.RemoveJob(jobID) {
			fmt.Printf("Removed job %s\n", jobID)
		} else {
			fmt.Printf("Job %s not found\n", jobID)
		}
	},
}

// cronEnableCmd represents the cron enable command
var cronEnableCmd = &cobra.Command{
	Use:   "enable [job-id]",
	Short: "Enable a job",
	Long:  `Enable a disabled job.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]
		disable, _ := cmd.Flags().GetBool("disable")

		// Load config to get store path
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dataDir := cfg.GetWorkspacePath() // Use workspace path for simplicity
		storePath := fmt.Sprintf("%s/data/cron/jobs.json", dataDir)

		// Create cron service
		service, err := cron.NewCronService(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cron service: %v\n", err)
			os.Exit(1)
		}

		job := service.EnableJob(jobID, !disable)
		if job != nil {
			status := "disabled"
			if !disable {
				status = "enabled"
			}
			fmt.Printf("Job '%s' %s\n", job.Name, status)
		} else {
			fmt.Printf("Job %s not found\n", jobID)
		}
	},
}

func init() {
	rootCmd.AddCommand(cronCmd)

	// Add subcommands
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronEnableCmd)

	// Cron list flags
	cronListCmd.Flags().Bool("all", false, "Include disabled jobs")

	// Cron add flags
	cronAddCmd.Flags().StringP("name", "n", "", "Job name (required)")
	cronAddCmd.Flags().StringP("message", "m", "", "Message for agent (required)")
	cronAddCmd.Flags().IntP("every", "e", 0, "Run every N seconds")
	cronAddCmd.Flags().StringP("cron", "c", "", "Cron expression (e.g. '0 9 * * *')")
	cronAddCmd.Flags().String("tz", "", "IANA timezone for cron (e.g. 'America/Vancouver')")
	cronAddCmd.Flags().String("at", "", "Run once at time (ISO format, e.g. '2026-02-12T10:30:00Z')")
	cronAddCmd.Flags().Bool("deliver", false, "Deliver response to channel")
	cronAddCmd.Flags().String("to", "", "Recipient for delivery")
	cronAddCmd.Flags().String("channel", "", "Channel for delivery (e.g. 'telegram', 'whatsapp')")

	// Mark required flags
	cronAddCmd.MarkFlagRequired("name")
	cronAddCmd.MarkFlagRequired("message")

	// Cron enable flags
	cronEnableCmd.Flags().Bool("disable", false, "Disable instead of enable")
}
