package main

import (
	"fmt"
	"os"
)

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: hi <api|sync|backtest|trading|notify|migrate|audit-prune>")
}
