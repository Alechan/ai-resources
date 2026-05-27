package main

import (
	"fmt"

	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"github.com/alejandro-danos/jenkinsctl/internal/service"
	"github.com/spf13/cobra"
)

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := jenkinsapi.New(url, user, token)
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
