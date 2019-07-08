package secrets

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"golang.org/x/crypto/openpgp"
)

func NewAmazonKeyProvider(pkSecretName, passphraseSecretName string, session *session.Session) *AmazonKeyProvider {
	return &AmazonKeyProvider{
		PrivateKeySecret: pkSecretName,
		PassphraseSecret: passphraseSecretName,
		secrets:          secretsmanager.New(session),
	}
}

type AmazonKeyProvider struct {
	PrivateKeySecret string
	PassphraseSecret string
	secrets          *secretsmanager.SecretsManager
}

func (provider *AmazonKeyProvider) GetBytesSecret(ctx context.Context, k string) ([]byte, error) {
	result, err := provider.secrets.GetSecretValueWithContext(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(k),
	})
	if err != nil {
		return nil, err
	}

	if result.SecretBinary == nil || len(result.SecretBinary) == 0 {
		return nil, errors.New("no SecretBinary in Amazon secret")
	}

	// Haven't figured out quite why yet but when creating a secret with the CLI somewhere
	// along the line a new-line character is added.
	return bytes.TrimRight(result.SecretBinary, "\n"), nil
}

// LoadPrivateKey obtains the GPG_PRIVATE_KEY from an AWS PrivateKeySecret and decrypts it using GPG_PASSPHRASE
func (provider *AmazonKeyProvider) LoadPrivateKey(ctx context.Context) (*openpgp.Entity, error) {
	key, err := provider.GetBytesSecret(ctx, provider.PrivateKeySecret)
	if err != nil {
		return nil, err
	}

	var passphrase []byte
	if provider.PassphraseSecret != "" {
		passphrase, err = provider.GetBytesSecret(ctx, provider.PassphraseSecret)
		if err != nil {
			return nil, err
		}
	}

	return DecryptKey(bytes.NewReader(key), passphrase)
}
