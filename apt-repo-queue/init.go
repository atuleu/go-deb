package main

import (
	"fmt"
	"os"
)

type InitCommand struct {
}

func (x *InitCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("I take no arguments")
	}

	err := os.MkdirAll(options.Base, 0755)
	if err != nil {
		return err
	}

	config, err := LoadConfig(options.Base)
	if err != nil {
		return err
	}

	if len(config.SignWith) == 0 {
		if err = config.PromptConfig(); err != nil {
			return err
		}
	}

	fmt.Printf("Saving config\n")
	err = config.Save()
	if err != nil {
		return err
	}

	keyManager, err := NewGpgKeyManager(config)
	if err != nil {
		return err
	}

	if len(keyManager.PrivateShortKeyID()) == 0 || len(config.SignWith) == 0 {
		fmt.Printf("Generating the private signing key, it could take some time\n")
		config, err = keyManager.SetupSignKey(config)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	parser.AddCommand("init",
		"System initialization command",
		"It will prompt the user for most configuration item, and initialiaze the system",
		&InitCommand{})
}
