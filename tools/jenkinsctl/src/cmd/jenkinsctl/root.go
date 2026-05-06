package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "jenkinsctl",
	Short: "Jenkins CLI wrapper",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var url string

func init() {
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "Jenkins URL")
}
