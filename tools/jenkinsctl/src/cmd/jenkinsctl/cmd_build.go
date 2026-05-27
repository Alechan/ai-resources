package main

import (
	"fmt"
	"io"

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
		client := jenkinsapi.New(url, user, token)
		svc := service.NewBuildService(client)
		build, err := svc.GetLastBuildStatus(jobName)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), formatBuildStatusOutput(build))
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

func formatBuildStatusOutput(build *service.Build) string {
	state := build.State()
	if state == "succeeded" {
		return fmt.Sprintf("state=%s build=%d", state, build.Number)
	}

	return fmt.Sprintf("state=%s build=%d url=%s", state, build.Number, build.URL)
}

func writeBuildStatus(out io.Writer, build *service.Build) error {
	_, err := fmt.Fprintln(out, formatBuildStatusOutput(build))
	return err
}
