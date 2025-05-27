package cmd

import (
	"fmt"
)

// VersionCmd shows version information
type VersionCmd struct{}

func (v *VersionCmd) Run(ctx *CLIContext) error {
	fmt.Printf("etcd2s3 %s\n", ctx.Version)
	return nil
}
