package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type AptPackage string

type AptDependencyTracker struct {
	packages []AptPackage
}

func (t *AptDependencyTracker) Add(p AptPackage) {
	t.packages = append(t.packages, p)
}

type packageStatus int

const (
	unreachable packageStatus = iota
	notInstalled
	installed
)

func getPackageStatus(p AptPackage) (packageStatus, error) {
	cmd := exec.Command("apt-cache", "policy", string(p))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return unreachable, fmt.Errorf("Could not check package `%s' status: ", err)
	}

	r := bufio.NewReader(strings.NewReader(string(output)))

	pReachable := false

	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return unreachable, fmt.Errorf("Could not parse apt-cache output while checking `%s' status: %s", p, err)
			}
			break
		}
		if strings.HasPrefix(l, "  Installed:") {
			pReachable = true
			l = strings.TrimSpace(strings.TrimPrefix(l, "  Installed:"))
			if l != "(none)" {
				return installed, nil
			}
		}
	}

	if pReachable {
		return notInstalled, nil
	}
	return unreachable, nil
}

func (t *AptDependencyTracker) SatisfyAll() error {
	toInstall := []AptPackage{}
	notReachable := []AptPackage{}
	for _, p := range t.packages {
		s, err := getPackageStatus(p)
		if err != nil {
			return err
		}
		switch s {
		case installed:
			continue
		case notInstalled:
			toInstall = append(toInstall, p)
		case unreachable:
			notReachable = append(notReachable, p)
		}
	}

	if len(notReachable) != 0 {
		return fmt.Errorf("Could not satisfy required apt dependencies %v, please check your apt sources", notReachable)
	}

	if len(toInstall) != 0 {
		cmd := exec.Command("sudo", "apt-get", "install", "-y")
		for _, p := range toInstall {
			cmd.Args = append(cmd.Args, string(p))
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		fmt.Printf("--- Installing packages %v, that will require super-user priviliege.\n", toInstall)
		fmt.Println(cmd.Args)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Could not install missing packages:  %s", err)
		}
	}

	return nil
}

var aptDepTracker = &AptDependencyTracker{}
