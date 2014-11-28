package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Getenv("GENERATE_MANPAGE")) != 0 {
		parser.WriteManPage(os.Stdout)
		os.Exit(0)
	}

	if _, err := parser.Parse(); err != nil {
		fmt.Fprintf(os.Stderr, "Got error: %s", err)
		os.Exit(1)
	}

}
