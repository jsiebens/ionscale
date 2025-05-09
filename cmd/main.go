package cmd

import (
	"github.com/jsiebens/ionscale/internal/cmd"
	"os"
)

func Main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
