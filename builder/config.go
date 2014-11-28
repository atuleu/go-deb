package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type ShowAllConfigCommand struct{}
type ShowConfigCommand struct{}
type SetConfigCommand struct{}
type EditConfigCommand struct{}

func (x *ShowAllConfigCommand) Execute(args []string) error {
	vOptions := reflect.ValueOf(*options)

	for i := 0; i < vOptions.NumField(); i = i + 1 {
		field := vOptions.Type().Field(i)

		jsonTag := field.Tag.Get("json")
		descTag := field.Tag.Get("description")
		if len(jsonTag) == 0 || jsonTag == "-" {
			continue
		}
		jsonTag = strings.Split(jsonTag, ",")[0]

		value := vOptions.Field(i).String()
		if len(value) == 0 {
			value = "%unset%"
		}
		fmt.Printf("- %s: %s #%s\n", jsonTag, value, descTag)
	}
	return nil
}

func (x *ShowConfigCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("config-show expects exactly one argument")
	}
	key := args[0]

	vOptions := reflect.ValueOf(*options)

	rx, err := regexp.Compile(key)
	if err != nil {
		return fmt.Errorf("Invalid search regexp `%s'", key)
	}

	found := 0
	for i := 0; i < vOptions.NumField(); i = i + 1 {
		field := vOptions.Type().Field(i)

		jsonTag := field.Tag.Get("json")
		descTag := field.Tag.Get("description")
		if len(jsonTag) == 0 || strings.HasPrefix(jsonTag, "-") {
			continue
		}
		jsonTag = strings.Split(jsonTag, ",")[0]

		if rx.MatchString(jsonTag) {
			value := vOptions.Field(i).String()
			if len(value) == 0 {
				value = "%unset%"
			}
			fmt.Printf("- %s: %s #%s\n", jsonTag, value, descTag)
			found = found + 1
		}
	}
	if found == 0 {
		fmt.Printf("No options matches regexp `%s'\n", key)
	} else {
		fmt.Printf("found %d options that matches regexp `%s'\n", found, key)
	}
	return nil
}

func (x *SetConfigCommand) Execute(args []string) error {

	for _, keyValue := range args {
		keyValueS := strings.Split(keyValue, "=")
		if len(keyValueS) != 2 {
			return fmt.Errorf("Invalid set syntax `%s', must be key=value", keyValue)
		}
		key := keyValueS[0]
		value := keyValueS[1]

		vOptions := reflect.ValueOf(options)
		found := false
		for i := 0; i < vOptions.Elem().NumField(); i = i + 1 {
			field := reflect.Indirect(vOptions).Type().Field(i)

			jsonTag := field.Tag.Get("json")
			if len(jsonTag) == 0 || strings.HasPrefix(jsonTag, "-") {
				continue
			}
			jsonTag = strings.Split(jsonTag, ",")[0]
			if jsonTag == key {
				vValue := vOptions.Elem().Field(i)
				if vValue.CanSet() == false {
					return fmt.Errorf("Internal error, cannot set field %v", key)
				}

				switch vValue.Kind() {
				case reflect.String:
					vValue.SetString(value)
				default:
					return fmt.Errorf("Cannot set options of Kind %v", vOptions.Field(i).Type())
				}
				found = true
				break
			}
		}
		if found == false {
			return fmt.Errorf("Did not found option `%s'", key)
		}
	}
	return options.Save()
}

func init() {

	parser.AddCommand("config-show-all",
		"Displays all settings on the command line",
		"Display all settings on the command line",
		&ShowAllConfigCommand{})

	parser.AddCommand("config-show",
		"Displays a list of settings on the command line",
		"It takes one arguments, which is a regexp of settings key",
		&ShowConfigCommand{})

	parser.AddCommand("config-set",
		"Setss a list of settings on the command line",
		"Sets the settings in the form key=value",
		&SetConfigCommand{})

}
