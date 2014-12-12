package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

type DebfileAuthentifier interface {
	CheckAnyClearsigned(r io.Reader) (io.Reader, error)
	SignChanges(filePath string) error
}

// A module that can read and check clearsigned content, and sign
// necessary dsc and control file
type GnupgAuthentifier struct {
	kr          openpgp.KeyRing
	pubringPath string
}

var gnupgHomeEnv = "GNUPGHOME"

// Creates a new authentifier that use the user gnupg data to check
// signature.
func NewAuthentifier() (*GnupgAuthentifier, error) {
	gnupghome := os.Getenv(gnupgHomeEnv)

	if len(gnupghome) == 0 {
		gnupghome = path.Join(os.Getenv("HOME"), ".gnupg")
	}

	res := &GnupgAuthentifier{}

	res.pubringPath = path.Join(gnupghome, "pubring.gpg")
	f, err := os.Open(res.pubringPath)
	if err != nil {
		return res, nil
	}
	defer f.Close()

	res.kr, err = openpgp.ReadKeyRing(f)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Reads a debian clearsigned file (.changes, .dsc) check the
// signature if any , and returns a plaintext version.
func (a *GnupgAuthentifier) CheckAnyClearsigned(r io.Reader) (io.Reader, error) {
	//we need to read all the file
	allData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	block, rest := clearsign.Decode(allData)
	if block == nil {
		return bytes.NewReader(rest), nil
	}

	if a.kr == nil {
		return nil, fmt.Errorf("data is clearsigned but no keyring file found in `%s'", a.pubringPath)
	}

	_, err = openpgp.CheckDetachedSignature(a.kr,
		bytes.NewReader(block.Bytes),
		block.ArmoredSignature.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(block.Plaintext), nil
}

func (a *GnupgAuthentifier) SignChanges(filePath string) error {
	if path.Ext(filePath) != ".changes" {
		return fmt.Errorf("invalid file to sign %s", filePath)
	}
	cmd := exec.Command("debsign", "-S", "--no-re-sign", filePath)
	//debsign should use ssh-agent, gnome-keyring-agent but not stdin
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not sign .changes file:\n%s", output)
	}
	return nil
}

func init() {
	aptDepTracker.Add("devscripts")
}
