package main

import (
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"github.com/alejandro-danos/jenkinsctl/internal/service"
	"github.com/spf13/cobra"
)

var buildStatusCmd = &cobra.Command{
	Use:   "status [job_name]",
	Short: "Get last build status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobName := args[0]
		cfg, err := app.LoadConfig()
		if err != nil {
			return err
		}
		client := jenkinsapi.New(url, auth.New(cfg))
		svc := service.NewBuildService(client)
		build, err := svc.GetLastBuildStatus(jobName)
		if err != nil {
			return err
		}
		fmt.Printf("Build #%d: %s\n", build.Number, build.Result)
		return nil
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage builds",
}

func init() {
	buildCmd.AddCommand(buildStatusCmd)
	rootCmd.AddCommand(buildCmd)
}
