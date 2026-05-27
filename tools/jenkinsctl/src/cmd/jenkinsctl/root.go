package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

var (
	url   string
	user  string
	token string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "Jenkins instance URL (required)")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "Jenkins username (required)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Jenkins API token (required)")
	_ = rootCmd.MarkPersistentFlagRequired("url")
	_ = rootCmd.MarkPersistentFlagRequired("user")
	_ = rootCmd.MarkPersistentFlagRequired("token")
}
