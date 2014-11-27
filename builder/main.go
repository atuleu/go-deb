package main

import "os"

func main() {
	options.LoadFromXDG()

	if len(os.Getenv("GENERATE_MANPAGE")) != 0 {
		parser.WriteManPage(os.Stdout)
		os.Exit(0)
	}

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

}
