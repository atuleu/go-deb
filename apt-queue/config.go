package main

import (
	"path"

	deb ".."
)

type Config struct {
	Base                                   string
	Label                                  string
	Origin                                 string
	Description                            string
	KeyType, KeyName, KeyComment, KeyEmail string
	KeySize                                int
	SignWith                               string
}

func (c *Config) Gnupghome() string {
	return path.Join(c.Base, ".gnupg")
}

func (c *Config) RepositoryPath() string {
	return path.Join(c.Base, "repository")
}

func (c *Config) Save() error {
	return deb.NotYetImplemented()
}
