// Lambda - Sign Package
// Receives a create object event from S3 in the form of un-signed RPM files
// it then signs the RPM file using a GPG key found in a given Amazon secret and uploads
// the signed RPM to a target bucket.
package main // import "git.illumina.com/relvacode/rpm-lambda/lambdas/sign-package"

import (
	"context"
	"errors"
	"fmt"
	"git.illumina.com/relvacode/rpm-lambda/events"
	"git.illumina.com/relvacode/rpm-lambda/secrets"
	"git.illumina.com/relvacode/rpm-lambda/setup"
	"git.illumina.com/relvacode/rpm-lambda/storage"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rustylynch/go-rpmutils"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvS3TargetBucket             = `LAMBDA_S3_TARGET`
	EnvS3TargetPath               = `LAMBDA_S3_TARGET_PATH`
	EnvS3BasePath                 = `LAMBDA_S3_BASE_PATH`
	EnvSigningKeySecret           = `LAMBDA_SECRET_GPG_KEY`
	EnvSigningKeyPassphraseSecret = `LAMBDA_SECRET_GPG_PASSPHRASE`
)

type LambdaFunction struct {
	l           aws.Logger
	s3          *storage.S3
	secrets     secrets.GPGProvider
	target      string
	target_path string
	base_path   string
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


	fmt.Printf("Received event for unsigned package in s3://%s/%s\n", event.Bucket.Name, event.Object.Key)
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


	fmt.Println("Package signed successfully.")

	s3Bucket := event.Bucket.Name
	if f.target != "" {
		s3Bucket = f.target
	}

	s3Key := event.Object.Key
	if f.target_path != "" {
		file_name := filepath.Base(event.Object.Key)
		file_path := strings.TrimLeft(filepath.Dir(f.target_path), "/")

		if f.base_path != "" {
			src_path := filepath.Dir(event.Object.Key)
			trim_base_path := strings.TrimLeft(filepath.Dir(f.base_path), "/")
			rel_path := strings.TrimLeft(strings.TrimPrefix(src_path, trim_base_path), "/")
			file_path = fmt.Sprintf("%s/%s", file_path, rel_path)
		}

		s3Key = fmt.Sprintf("%s/%s", file_path, file_name)
	}

	fmt.Printf("Moving signed package to s3://%s/%s\n", s3Bucket, s3Key)

	err = f.s3.UploadObject(groupCtx, pr, s3Bucket, s3Key, "application/x-rpm")
	if err != nil {
		return err
	}
	_ = pr.Close()

	err = g.Wait()
	if err != nil {
		return err
	}

	fmt.Println("Signed package uploaded successfully.")

	// finally, delete the original object
	err = f.s3.DeleteObject(ctx, event.Bucket.Name, event.Object.Key)
	if err != nil {
		return err
	}

	fmt.Println("Original package deleted successfully.")

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

		target := setup.GetEnv(EnvS3TargetBucket, "")
		target_path := setup.GetEnv(EnvS3TargetPath, "")
		base_path := setup.GetEnv(EnvS3BasePath, "")

		if target == "" && target_path == "" {
			return errors.New(fmt.Sprintf("Either %s or %s must be set.", EnvS3TargetBucket, EnvS3TargetPath))
		}

		if base_path != "" && target_path == "" {
			return errors.New(fmt.Sprintf("Setting %s also requires %s to be set.", EnvS3BasePath, EnvS3TargetPath))
		}

		f := LambdaFunction{
			target:      target,
			target_path: target_path,
			base_path:   base_path,
			l:           setup.NewLog("lambda:sign-repo"),
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
