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
	SetupSignKey(conf *Config) (*Config, error)
}

type GpgKeyManager struct {
	public_keys openpgp.EntityList
	secret_keys openpgp.EntityList
	masterKeyId string
	gnupghome   string
}

func NewGpgKeyManager(conf *Config) (*GpgKeyManager, error) {
	res := &GpgKeyManager{
		gnupghome:   conf.Gnupghome(),
		masterKeyId: conf.SignWith,
	}
	err := os.MkdirAll(conf.Gnupghome(), 0700)
	if err != nil {
		return nil, err
	}
	err = res.load()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m *GpgKeyManager) load() error {
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

	if len(m.masterKeyId) > 0 {
		if len(m.secret_keys) == 0 {
			return fmt.Errorf("There is no private keyring")
		} else if len(m.secret_keys) > 1 {
			return fmt.Errorf("There is more than one private key")
		}

		keyId := m.PrivateShortKeyID()
		if m.masterKeyId != keyId {
			return fmt.Errorf("Configuration ask for private key %s, but only %s is accessible", m.masterKeyId, keyId)
		}

	}

	return nil
}

func (m *GpgKeyManager) GpgCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("gpg", args...)

	cmd.Env = []string{fmt.Sprintf("GNUPGHOME=%s", m.gnupghome)}

	return cmd
}

func (m *GpgKeyManager) Add(r io.Reader) error {
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
	cmd := m.GpgCommand("--import", f.Name())
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

func (m *GpgKeyManager) List() openpgp.EntityList {
	res := make(openpgp.EntityList, 0)

	encryptionKeyId := m.PrivateShortKeyID()
	for _, k := range m.public_keys {
		if k.PrimaryKey.KeyIdShortString() == encryptionKeyId {
			continue
		}
		res = append(res, k)
	}
	return res
}

func (m *GpgKeyManager) Remove(keyId string) error {
	cmd := m.GpgCommand("--delete-key", keyId)
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

func (m *GpgKeyManager) CheckAndRemoveClearsigned(r io.Reader) (io.Reader, error) {
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

func (m *GpgKeyManager) PrivateShortKeyID() string {
	if len(m.secret_keys) != 1 {
		return ""
	}
	return m.secret_keys[0].PrivateKey.KeyIdShortString()
}

func (m *GpgKeyManager) SetupSignKey(conf *Config) (*Config, error) {
	if len(conf.SignWith) != 0 {
		if len(m.PrivateShortKeyID()) != 0 && conf.SignWith != m.PrivateShortKeyID() {
			return nil, fmt.Errorf("Cannot setup signing key, as I have already a signing key %s, and configuration want %s",
				m.PrivateShortKeyID(),
				conf.SignWith)
		}
	}

	if len(m.secret_keys) == 0 {

		cmd := m.GpgCommand("--batch", "-v", "--gen-key")

		var in bytes.Buffer
		cmd.Stdin = &in
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		sync := make(chan error)
		go func() {
			sync <- cmd.Run()
		}()

		fmt.Fprintf(&in, "Key-Type: %s\n", conf.KeyType)
		fmt.Fprintf(&in, "Key-Length: %d\n", conf.KeySize)
		fmt.Fprintf(&in, "Name-Real: %s\n", conf.KeyName)
		fmt.Fprintf(&in, "Name-Comment: %s\n", conf.KeyComment)
		fmt.Fprintf(&in, "Name-Email: %s\n%%commit\n", conf.KeyEmail)

		err := <-sync

		if err != nil {
			return nil, fmt.Errorf("Could not generate signing key: %s", err)
		}

		err = m.load()
		if err != nil {
			return nil, err
		}

	}

	conf.SignWith = m.PrivateShortKeyID()
	err := conf.Save()
	if err != nil {
		return nil, err
	}
	return conf, nil
}
