package secrets

import (
	"golang.org/x/crypto/openpgp"
	"os"
)

type FilepathGPGProvider struct {
	Filepath   string
	Passphrase []byte
}

func (cs *FilepathGPGProvider) LoadPrivateKey() (*openpgp.Entity, error) {
	f, err := os.OpenFile(cs.Filepath, os.O_RDONLY, os.FileMode(0600))
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return DecryptKey(f, cs.Passphrase)
}
