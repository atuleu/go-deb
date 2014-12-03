package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

// A module that can read and check clearsigned content, and sign
// necessary dsc and control file
type DebfileAuthentifier struct {
	kr          openpgp.KeyRing
	pubringPath string
}

var gnupgHomeEnv = "GNUPGHOME"

// Creates a new authentifier that use the user gnupg data to check
// signature.
func NewAuthentifier() (*DebfileAuthentifier, error) {
	gnupghome := os.Getenv(gnupgHomeEnv)

	if len(gnupghome) == 0 {
		gnupghome = path.Join(os.Getenv("HOME"), ".gnupg")
	}

	res := &DebfileAuthentifier{}

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
func (a *DebfileAuthentifier) CheckAnyClearsigned(r io.Reader) (io.Reader, error) {
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

func init() {
	aptDepTracker.Add("devscripts")
}
