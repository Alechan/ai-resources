package main

import (
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"github.com/alejandro-danos/jenkinsctl/internal/service"
	"github.com/spf13/cobra"
)

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := app.LoadConfig()
		if err != nil {
			return err
		}
		client := jenkinsapi.New(url, auth.New(cfg))
		svc := service.NewJobService(client)
		jobs, err := svc.ListJobs()
		if err != nil {
			return err
		}
		for _, j := range jobs {
			fmt.Printf("%s (%s)\n", j.Name, j.Color)
		}
		return nil
	},
}

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage jobs",
}

func init() {
	jobCmd.AddCommand(jobListCmd)
	rootCmd.AddCommand(jobCmd)
}
