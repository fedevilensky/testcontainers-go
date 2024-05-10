package cmd

import (
	"github.com/spf13/cobra"

	"github.com/testcontainers/testcontainers-go/devtools/cmd/modules"
	"github.com/testcontainers/testcontainers-go/devtools/cmd/release"
)

var NewRootCmd = &cobra.Command{
	Use:   "devtools",
	Short: "Management tool for testcontainers-go",
	Long:  "Management tool for testcontainers-go",
}

func init() {
	NewRootCmd.AddCommand(modules.NewCmd)
	NewRootCmd.AddCommand(release.ReleaseCmd)
}
