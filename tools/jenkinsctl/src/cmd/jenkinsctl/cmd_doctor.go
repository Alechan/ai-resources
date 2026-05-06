package main

import (
	"fmt"
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"github.com/alejandro-danos/jenkinsctl/internal/auth"
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"github.com/alejandro-danos/jenkinsctl/internal/service"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := app.LoadConfig()
		if err != nil {
			return err
		}
		client := jenkinsapi.New(url, auth.New(cfg))
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
