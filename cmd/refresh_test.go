package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRefreshFlags(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		command string
		watch   bool
		invalid bool
	}{
		{name: "stdin"},
		{name: "manual", args: []string{"--refresh"}, command: "git diff"},
		{name: "short manual", args: []string{"-r"}, command: "git diff"},
		{name: "custom manual", args: []string{"--refresh-cmd", "git diff HEAD"}, command: "git diff HEAD"},
		{name: "watch", args: []string{"--watch"}, command: "git diff", watch: true},
		{name: "custom watch", args: []string{"--watch-cmd", "git diff --cached"}, command: "git diff --cached", watch: true},
		{name: "conflicting modes", args: []string{"--refresh", "--watch"}, invalid: true},
		{name: "manual with watch command", args: []string{"--refresh", "--watch-cmd", "git diff"}, invalid: true},
		{name: "watch with manual command", args: []string{"--watch", "--refresh-cmd", "git diff"}, invalid: true},
		{name: "conflicting commands", args: []string{"--refresh-cmd", "git diff", "--watch-cmd", "git diff"}, invalid: true},
		{name: "conflicting PR", args: []string{"--refresh", "--pr", "https://github.com/a/b/pull/1"}, invalid: true},
		{name: "empty command", args: []string{"--refresh-cmd", " "}, invalid: true},
		{name: "zero interval", args: []string{"--watch", "--watch-interval", "0s"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("pr", "", "")
			addRefreshFlags(cmd)
			if err := cmd.ParseFlags(tc.args); err != nil {
				t.Fatal(err)
			}
			cfg, err := refreshConfig(cmd)
			if err == nil {
				err = cmd.ValidateFlagGroups()
			}
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v, invalid = %v", err, tc.invalid)
			}
			if !tc.invalid &&
				(cfg.Cmd != tc.command || cfg.Watch != tc.watch || cfg.Interval != 2*time.Second) {
				t.Fatalf("unexpected config: %+v", cfg)
			}
		})
	}
}
