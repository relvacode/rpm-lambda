package secrets

import (
	"context"
	"errors"
	"golang.org/x/crypto/openpgp"
	"io"
)

var (
	ErrSecretsInvalidKeyring    = errors.New("secrets: expected exactly one entity in private key keyring")
	ErrSecretsMissingPassphrase = errors.New("secrets: passphrase is required with an encrypted private key")
)

type GPGProvider interface {
	LoadPrivateKey(ctx context.Context) (*openpgp.Entity, error)
}

// DetachedSign signs data from `r` using `key` into the signature stream `w`.
func DetachedSign(key *openpgp.Entity, r io.Reader, w io.Writer) error {
	return openpgp.DetachSign(w, key, r, nil)
}

func DecryptKey(key io.Reader, passphrase []byte) (*openpgp.Entity, error) {
	keyring, err := openpgp.ReadArmoredKeyRing(key)
	if err != nil {
		return nil, err
	}

	if len(keyring) != 1 {
		return nil, ErrSecretsInvalidKeyring
	}

	entity := keyring[0]
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		if len(passphrase) == 0 {
			return nil, ErrSecretsMissingPassphrase
		}
		err = entity.PrivateKey.Decrypt(passphrase)
		if err != nil {
			return nil, err
		}
	}
	return entity, nil
}
