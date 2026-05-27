package main

import (
	"fmt"

	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"github.com/alejandro-danos/jenkinsctl/internal/service"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := jenkinsapi.New(url, user, token)
		svc := service.NewDoctorService(client)
		if err := svc.CheckConnectivity(); err != nil {
			return err
		}
		fmt.Println("Successfully connected to Jenkins.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
