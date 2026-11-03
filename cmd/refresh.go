package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dlvhdr/diffnav/pkg/config"
)

func addRefreshFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("refresh", "r", false, "Run git diff and refresh manually with r")
	cmd.Flags().String("refresh-cmd", "", "Command to run initially and on r (implies --refresh)")
	cmd.Flags().
		BoolP("watch", "w", false, "Watch mode: periodically re-run a diff command and refresh")
	cmd.Flags().String("watch-cmd", "git diff", "Command to run in watch mode")
	cmd.Flags().Duration("watch-interval", 2*time.Second, "Interval between watch refreshes")
	cmd.MarkFlagsMutuallyExclusive("refresh", "watch")
	cmd.MarkFlagsMutuallyExclusive("refresh", "watch-cmd")
	cmd.MarkFlagsMutuallyExclusive("refresh-cmd", "watch")
	cmd.MarkFlagsMutuallyExclusive("refresh-cmd", "watch-cmd")
	for _, flag := range []string{"refresh", "refresh-cmd", "watch", "watch-cmd"} {
		cmd.MarkFlagsMutuallyExclusive("pr", flag)
	}
}

func refreshConfig(cmd *cobra.Command) (config.RefreshConfig, error) {
	flags := cmd.Flags()
	manual, _ := flags.GetBool("refresh")
	watch, _ := flags.GetBool("watch")
	interval, _ := flags.GetDuration("watch-interval")
	watch = watch || flags.Changed("watch-cmd")
	cfg := config.RefreshConfig{Watch: watch, Interval: interval}
	switch {
	case watch:
		cfg.Cmd, _ = flags.GetString("watch-cmd")
	case flags.Changed("refresh-cmd"):
		cfg.Cmd, _ = flags.GetString("refresh-cmd")
	case manual:
		cfg.Cmd = "git diff"
	default:
		return cfg, nil
	}
	if strings.TrimSpace(cfg.Cmd) == "" {
		return cfg, fmt.Errorf("refresh command must not be empty")
	}
	if watch && interval <= 0 {
		return cfg, fmt.Errorf("watch interval must be greater than zero")
	}
	return cfg, nil
}
