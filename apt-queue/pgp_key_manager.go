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

type PgpKeyManager interface {
	Add(r io.Reader) error
	List() openpgp.EntityList
	Remove(string) error
	CheckAndRemoveClearsigned(r io.Reader) (io.Reader, error)
	PrivateShortKeyID() string
	SetupPrivate(keyType, name, comment, address string, size int) error
}

type HomeGpgKeyManager struct {
	public_keys openpgp.KeyRing
	secret_keys openpgp.KeyRing
	gnupghome   string
}

func NewHomeGpgKeyManager() (*HomeGpgKeyManager, error) {
	res := &HomeGpgKeyManager{}
	res.gnupghome = os.Getenv("GNUPGHOME")
	if len(res.gnupghome) == 0 {
		home := os.Getenv("HOME")
		if len(home) == 0 {
			return nil, fmt.Errorf("Could not define GNUPGHOME")
		}
		res.gnupghome = path.Join(home, ".gnupg")
	}

	err := res.load()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m *HomeGpgKeyManager) load() error {
	f, err := os.Open(path.Join(m.gnupghome, "pubring.gpg"))
	if err != nil && os.IsNotExist(err) == false {
		return err
	}
	if err == nil {
		m.public_keys, err = openpgp.ReadKeyRing(f)

		if err != nil {
			return err
		}
	}

	f, err = os.Open(path.Join(m.gnupghome, "secring.gpg"))
	if err != nil && os.IsNotExist(err) == false {
		return err
	}
	if err == nil {
		m.secret_keys, err = openpgp.ReadKeyRing(f)
		if err != nil {
			return err
		}
	}

	if m.secret_keys != nil {
		if len(m.secret_keys.DecryptionKeys()) != 1 {
			return fmt.Errorf("Should have only one, unsigned secret key")
		}
	}

	return nil
}

func (m *HomeGpgKeyManager) Add(r io.Reader) error {
	var toTest, toCopy bytes.Buffer
	_, err := io.Copy(io.MultiWriter(&toTest, &toCopy), r)
	if err != nil {
		return err
	}
	_, err = openpgp.ReadArmoredKeyRing(&toTest)
	if err != nil {
		return fmt.Errorf("is not a pgp armored keyring: %s", err)
	}

	f, err := ioutil.TempFile("", "keyring-adding")
	if err != nil {
		return err
	}
	_, err = io.Copy(f, &toCopy)
	if err != nil {
		return err
	}
	cmd := exec.Command("gpg", "--import", f.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not add key: %s\n%s", err, out)
	}
	err = m.load()
	if err != nil {
		return err
	}
	return nil
}

func (m *HomeGpgKeyManager) List() openpgp.EntityList {
	res := make(openpgp.EntityList, 0)

	encryptioneyId := m.PrivateShortKeyID()
	for _, k := range m.public_keys.DecryptionKeys() {
		if k.PublicKey.KeyIdShortString() == encryptioneyId {
			continue
		}
		res = append(res, k.Entity)
	}
	return res
}

func (m *HomeGpgKeyManager) Remove(keyId string) error {
	cmd := exec.Command("gpg", "--delete-key", keyId)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not remove key %s : %s\n%s", keyId, err, out)
	}
	err = m.load()
	if err != nil {
		return err
	}
	return nil
}

func (m *HomeGpgKeyManager) CheckAndRemoveClearsigned(r io.Reader) (io.Reader, error) {
	data, err := ioutil.ReadAll(r)
	block, rest := clearsign.Decode(data)

	if block == nil {
		return bytes.NewReader(rest), fmt.Errorf("File is not clearsigned with a signature")
	}

	res := bytes.NewReader(block.Plaintext)
	_, err = openpgp.CheckDetachedSignature(m.public_keys,
		bytes.NewReader(block.Bytes),
		block.ArmoredSignature.Body)

	return res, err
}

func (m *HomeGpgKeyManager) PrivateShortKeyID() string {
	if len(m.secret_keys.DecryptionKeys()) != 1 {
		return ""
	}
	return m.secret_keys.DecryptionKeys()[0].PublicKey.KeyIdShortString()
}

func (m *HomeGpgKeyManager) SetupPrivate(keyType, name, comment, address string, size int) error {
	if len(m.PrivateShortKeyID()) != 0 {
		return fmt.Errorf("I already have a signing key")
	}

	cmd := exec.Command("gpg", "--batch", "--no-tty", "--gen-key")

	var in bytes.Buffer
	cmd.Stdin = &in
	fmt.Fprintf(&in, "Key-Type: %s\n", keyType)
	fmt.Fprintf(&in, "Key-Length: %d\n", size)
	fmt.Fprintf(&in, "Name-Real: %s\n", name)
	fmt.Fprintf(&in, "Name-Comment: %s\n", comment)
	fmt.Fprintf(&in, "Name-Email: %s\n%%commit\n", address)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not generate signing key: %s", out)
	}

	err = m.load()
	if err != nil {
		return err
	}

	return nil
}
