package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	Base        string
	Label       string `question:"Please enter the label of the repository"`
	Origin      string `question:"Please enter the origin of the repository"`
	Description string `question:"Please enter the description of the repository"`
	KeyType     string `question:"Please enter the type of the publishing key"`
	KeyName     string `question:"Please enter the name of the publishing key"`
	KeyComment  string `question:"Please enter the comment of the publishing key"`
	KeyEmail    string `question:"Please enter the email of the publishing key"`
	KeySize     int    `question:"Please enter the size of the publishing key"`
	SignWith    string
}

func (c *Config) Gnupghome() string {
	return path.Join(c.Base, ".gnupg")
}

func (c *Config) RepositoryPath() string {
	return path.Join(c.Base, "repository")
}

func (c *Config) ConfPath() string {
	return path.Join(c.Base, "config.json")
}

func (c *Config) Save() error {
	f, err := os.Create(c.ConfPath())
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)

	return enc.Encode(c)
}

func LoadConfig(base string) (*Config, error) {
	config := &Config{
		Base:    base,
		KeyType: "RSA",
		KeySize: 2048,
	}
	f, err := os.Open(config.ConfPath())
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) PromptConfig() error {
	cValue := reflect.ValueOf(c).Elem()

	cType := reflect.TypeOf(c).Elem()
	for i := 0; i < cType.NumField(); i = i + 1 {
		fType := cType.Field(i)
		question := fType.Tag.Get("question")
		if len(question) == 0 {
			continue
		}
		fValue := cValue.FieldByName(fType.Name)
		switch fValue.Kind() {
		case reflect.String:
			fmt.Printf("%s[%s]:", question, fValue.String())
			r := bufio.NewReader(os.Stdin)
			res, err := r.ReadString('\n')
			if err != nil {
				return err
			}
			if res != "\n" {
				fValue.SetString(strings.TrimSpace(res))
			}
		case reflect.Int:
			fmt.Printf("%s[%d]:", question, fValue.Int())
			r := bufio.NewReader(os.Stdin)
			res, err := r.ReadString('\n')
			if err != nil {
				return err
			}
			if res != "\n" {
				resInt, err := strconv.ParseInt(strings.TrimSpace(res), 0, 64)
				if err != nil {
					return err
				}
				fValue.SetInt(resInt)
			}
		default:
			return fmt.Errorf("Could not prompt value of kind %v", fValue.Kind())
		}
	}

	return nil
}
