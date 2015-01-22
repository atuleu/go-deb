package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"

	"golang.org/x/crypto/openpgp"
)

var ppaAddressRx = regexp.MustCompile(`^ppa:([a-z0-9][a-zA-Z0-9\.\+\-]*)/([a-z0-9][a-zA-Z\.\-\+]*)$`)
var ppaApiRequestFmt = "http://api.launchpad.net/1.0/~%s/+archive/%s"
var ppaAptRepoAddressFmt = "http://ppa.launchpad.net/%s/%s/ubuntu"

type jsonPPAData struct {
	SigningKeyFingerprint string `json:"signing_key_fingerprint"`
}

func fetchArmoredKey(fingerprint string, server string) ([]byte, error) {
	tmpDir, err := ioutil.TempDir("", "gpg_key_fetcher")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	pubringPath := path.Join(tmpDir, "pubring.gpg")
	secringPath := path.Join(tmpDir, "secring.gpg")
	for _, p := range []string{pubringPath, secringPath} {
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	cmd := exec.Command("gpg", "--no-default-keyring", "--no-options",
		"--homedir", tmpDir,
		"--secret-keyring", secringPath,
		"--keyring", pubringPath,
		"--keyserver", server,
		"--recv", fingerprint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Could not fetch key %s from %s: %s\n%s", fingerprint, server, err, out)
	}
	cmd = exec.Command("gpg", "--no-default-keyring", "--no-options",
		"--homedir", tmpDir,
		"--keyring", pubringPath,
		"--armor",
		"--export", fingerprint)
	var armoredData, cmdOut bytes.Buffer
	cmd.Stdout = &armoredData
	cmd.Stderr = &cmdOut
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Could not armor keydata: %s\n%s", err, cmdOut.String())
	}

	//verify the received key data
	keys, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(armoredData.Bytes()))
	if err != nil {
		return nil, err
	}

	if len(keys) != 1 {
		return nil, fmt.Errorf("Received more than one key from server.")
	}
	receivedFingerprint := fmt.Sprintf("%X", keys[0].PrimaryKey.Fingerprint)
	if receivedFingerprint != fingerprint {
		return nil, fmt.Errorf("Invalid received key fingerprint %s, expected %s", receivedFingerprint, fingerprint)
	}

	return armoredData.Bytes(), nil
}

func NewPPArepositoryAccess(address string) (*AptRepositoryAccess, error) {
	matches := ppaAddressRx.FindStringSubmatch(address)
	if matches == nil {
		return nil, fmt.Errorf("Invalid PPA address %s", address)
	}

	owner := matches[1]
	ppaName := matches[2]

	res := &AptRepositoryAccess{
		ID:      AptRepositoryID(address),
		Address: fmt.Sprintf(ppaAptRepoAddressFmt, owner, ppaName),
	}

	resp, err := http.Get(fmt.Sprintf(ppaApiRequestFmt, owner, ppaName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	data := jsonPPAData{}
	err = dec.Decode(&data)
	if err != nil {
		return nil, err
	}

	res.ArmoredPublicKey, err = fetchArmoredKey(data.SigningKeyFingerprint,
		"hkp://keyserver.ubuntu.com:80")
	if err != nil {
		return nil, err
	}
	return res, nil
}
