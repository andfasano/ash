package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/ash/pkg/ash"
)

func main() {

	var command string

	if len(os.Args) < 2 {
		fmt.Println("Please specify at least one command, setup or teardown")
		os.Exit(1)
	}
	command = os.Args[1]

	cacheDir, _ := os.UserCacheDir()
	ash := ash.NewAgentScenarioHelper(filepath.Join(cacheDir, "ash"))

	var err error
	switch command {
	case "setup":
		err = ash.Setup()
	case "teardown":
		err = ash.Teardown()
	default:
		fmt.Println("Unknown command")
		os.Exit(1)
	}
	if err != nil {
		fmt.Println(err)
	}
}
