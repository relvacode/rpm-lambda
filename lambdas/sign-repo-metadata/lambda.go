// Lambda - Sign Repo Metadata
// Receives a create object request from S3 in the form of repository metadata files (.xml and .xml.gz)
// then signs the contents of each file and uploads the signature (.asc) to the same bucket as the
// originating request
package main // import "git.illumina.com/relvacode/rpm-lambda/lambdas/sign-repo"

import (
	"bytes"
	"context"
	"fmt"
	"git.illumina.com/relvacode/rpm-lambda/events"
	"git.illumina.com/relvacode/rpm-lambda/secrets"
	"git.illumina.com/relvacode/rpm-lambda/setup"
	"git.illumina.com/relvacode/rpm-lambda/storage"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"golang.org/x/crypto/openpgp"
)

const (
	EnvSigningKeySecret           = `LAMBDA_SECRET_GPG_KEY`
	EnvSigningKeyPassphraseSecret = `LAMBDA_SECRET_GPG_PASSPHRASE`
)

type LambdaFunction struct {
	l       aws.Logger
	s3      *storage.S3
	secrets secrets.GPGProvider
}

func (f *LambdaFunction) HandleEvent(ctx context.Context, key *openpgp.Entity, event events.Event) error {
	var b bytes.Buffer

	_, r, err := f.s3.DownloadObject(ctx, event.Bucket.Name, event.Object.Key)
	if err != nil {
		return err
	}

	err = secrets.DetachedSign(key, r, &b)
	_ = r.Close()
	if err != nil {
		return err
	}

	err = f.s3.UploadObject(ctx, &b, event.Bucket.Name, fmt.Sprintf("%s.asc", event.Object.Key), "application/octet-stream")
	if err != nil {
		return err
	}

	return nil
}

func (f *LambdaFunction) HandleRequest(ctx context.Context, topic *events.LambdaS3CreateObjectEvent) error {
	key, err := f.secrets.LoadPrivateKey(ctx)
	if err != nil {
		return err
	}

	for _, e := range topic.Events() {
		err = f.HandleEvent(ctx, key, e)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	setup.Main(func() error {
		s, err := setup.NewSession()
		if err != nil {
			return err
		}

		f := LambdaFunction{
			l: setup.NewLog("lambda:sign-repo"),
			s3: &storage.S3{
				Session: s,
			},
			secrets: secrets.NewAmazonKeyProvider(
				setup.GetEnv(EnvSigningKeySecret),
				setup.GetEnv(EnvSigningKeyPassphraseSecret, ""),
				s),
		}

		lambda.Start((&f).HandleRequest)
		return nil
	})
}
