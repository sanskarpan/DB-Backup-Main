package commands

import (
	"time"

	"github.com/spf13/cobra"
)

// flagString returns the string value of the named flag, or "" if it cannot be read.
func flagString(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}
	return v
}

// flagInt returns the int value of the named flag, or 0 if it cannot be read.
func flagInt(cmd *cobra.Command, name string) int {
	v, err := cmd.Flags().GetInt(name)
	if err != nil {
		return 0
	}
	return v
}

// flagBool returns the bool value of the named flag, or false if it cannot be read.
func flagBool(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}
	return v
}

// flagStringSlice returns the string slice value of the named flag, or nil if it cannot be read.
func flagStringSlice(cmd *cobra.Command, name string) []string {
	v, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil
	}
	return v
}

// flagDuration returns the duration value of the named flag, or 0 if it cannot be read.
func flagDuration(cmd *cobra.Command, name string) time.Duration {
	v, err := cmd.Flags().GetDuration(name)
	if err != nil {
		return 0
	}
	return v
}
