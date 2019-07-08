// Lambda - Sign Package
// Receives a create object event from S3 in the form of un-signed RPM files
// it then signs the RPM file using a GPG key found in a given Amazon secret and uploads
// the signed RPM to a target bucket.
package main // import "git.illumina.com/relvacode/rpm-lambda/lambdas/sign-package"

import (
	"context"
	"git.illumina.com/relvacode/rpm-lambda/events"
	"git.illumina.com/relvacode/rpm-lambda/secrets"
	"git.illumina.com/relvacode/rpm-lambda/setup"
	"git.illumina.com/relvacode/rpm-lambda/storage"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/sassoftware/go-rpmutils"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	EnvS3TargetBucket             = `LAMBDA_S3_TARGET`
	EnvSigningKeySecret           = `LAMBDA_SECRET_GPG_KEY`
	EnvSigningKeyPassphraseSecret = `LAMBDA_SECRET_GPG_PASSPHRASE`
)

type LambdaFunction struct {
	l       aws.Logger
	s3      *storage.S3
	secrets secrets.GPGProvider
	target  string
}

func (f *LambdaFunction) HandleEvent(ctx context.Context, key *openpgp.Entity, event events.Event) error {
	if !strings.HasSuffix(event.Object.Key, ".rpm") {
		return nil
	}

	// Open a temporary file to write signed RPM contents
	// (signing an RPM requires a read-seeker)
	fd, err := ioutil.TempFile(os.TempDir(), "rpm-sign")
	if err != nil {
		return err
	}

	defer os.Remove(fd.Name())
	defer fd.Close()

	_, r, err := f.s3.DownloadObject(ctx, event.Bucket.Name, event.Object.Key)
	if err != nil {
		return err
	}

	// copy the contents of the un-signed RPM file to disk
	_, err = io.Copy(fd, r)
	if err != nil {
		return err
	}

	err = r.Close()
	if err != nil {
		return err
	}

	// seek to the beginning of the file
	_, err = fd.Seek(0, 0)
	if err != nil {
		return err
	}

	pr, pw := io.Pipe()

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err = rpmutils.SignRpmFileIntoStream(pw, fd, key.PrivateKey, nil)
		_ = pw.CloseWithError(err)
		return err
	})

	err = f.s3.UploadObject(groupCtx, pr, f.target, event.Object.Key, "application/x-rpm")
	if err != nil {
		return err
	}
	_ = pr.Close()

	err = g.Wait()
	if err != nil {
		return err
	}

	// finally, delete the original object
	err = f.s3.DeleteObject(ctx, event.Bucket.Name, event.Object.Key)
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
			target: setup.GetEnv(EnvS3TargetBucket),
			l:      setup.NewLog("lambda:sign-repo"),
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
