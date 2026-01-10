package main

import (
  "os"
  "restorable.io/restorable-cli/internal/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
	os.Exit(3) // CLI/config error
    }
}
