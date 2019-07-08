// Lambda - Create Repo Metadata
// Receives S3 create object requests via a SQS queue and processes each RPM file
// found within the request by updating repository metadata in the same S3 bucket as the originating request
package main // import "git.illumina.com/relvacode/rpm-lambda/lambdas/create-repo-metadata"

import (
	"context"
	"encoding/xml"
	"git.illumina.com/relvacode/rpm-lambda/events"
	"git.illumina.com/relvacode/rpm-lambda/setup"
	"git.illumina.com/relvacode/rpm-lambda/storage"
	"git.illumina.com/relvacode/rpm-lambda/yum"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"strings"
)

type LambdaFunction struct {
	l  aws.Logger
	s3 *storage.S3
}

func (f *LambdaFunction) GetRepository(ctx context.Context, bucket string) (*yum.Repository, error) {
	metadata := yum.MetadataData{
		XMLName: xml.Name{
			Local: "repomd",
		},
	}

	_, err := f.s3.DownloadXMLObject(ctx, &metadata, bucket, storage.RepoMDXML)
	if err != nil {
		return nil, err
	}

	filelist := yum.FilelistData{
		XMLName: xml.Name{
			Local: "filelists",
		},
	}

	_, err = f.s3.DownloadCompressedXMLObject(ctx, &filelist, bucket, storage.FilelistXML)
	if err != nil {
		return nil, err
	}

	packages := yum.PackageData{
		XMLName: xml.Name{
			Local: "metadata",
		},
	}

	_, err = f.s3.DownloadCompressedXMLObject(ctx, &packages, bucket, storage.PrimaryXML)
	if err != nil {
		return nil, err
	}

	return &yum.Repository{
		Metadata: &metadata,
		Filelist: &filelist,
		Packages: &packages,
	}, nil
}

func (f *LambdaFunction) PutRepository(ctx context.Context, bucket string, repo *yum.Repository) error {
	primary, err := f.s3.UploadCompressedXMLObject(ctx, repo.Packages, bucket, storage.PrimaryXML)
	if err != nil {
		return err
	}

	// Regenerate package data check-sums
	repo.Metadata.Update(storage.PrimaryXMLObject{XMLObject: *primary}.Metadata())

	filelist, err := f.s3.UploadCompressedXMLObject(ctx, repo.Filelist, bucket, storage.FilelistXML)
	if err != nil {
		return err
	}

	// Regenerate filelist data check-sums
	repo.Metadata.Update(storage.FilelistXMLObject{XMLObject: *filelist}.Metadata())

	err = f.s3.UploadXMLObject(ctx, repo.Metadata, bucket, storage.RepoMDXML)
	if err != nil {
		return err
	}

	return nil
}

func (f *LambdaFunction) LoadRPM(ctx context.Context, r events.Event) (*yum.RPMObject, error) {
	o, err := s3.New(f.s3).GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &r.Bucket.Name,
		Key:    &r.Object.Key,
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed downloading RPM object")
	}

	rpm, err := yum.ScanRPM(ctx, o.Body)
	_ = o.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan RPM")
	}

	return &yum.RPMObject{
		RPM: *rpm,
		Key: r.Object.Key,
	}, err
}

func (f *LambdaFunction) HandleBucketRequest(ctx context.Context, bucket string, events []events.Event) error {
	var packages []*yum.RPMObject
	for _, record := range events {
		// skip non-RPM files
		if !strings.HasSuffix(record.Object.Key, ".rpm") {
			continue
		}
		rpm, err := f.LoadRPM(ctx, record)
		if err != nil {
			return err
		}
		packages = append(packages, rpm)
	}

	if len(packages) == 0 {
		return nil
	}

	repository, err := f.GetRepository(ctx, bucket)
	if err != nil {
		return err
	}

	updated := repository.Update(packages...)
	if !updated {
		return nil
	}

	err = f.PutRepository(ctx, bucket, repository)
	if err != nil {
		return err
	}

	return nil
}

func (f *LambdaFunction) HandleRequest(ctx context.Context, topic *events.SQSUpdateRepoEvent) error {
	mapping, err := topic.Events()
	if err != nil {
		return err
	}

	for bucket, ev := range mapping {
		err = f.HandleBucketRequest(ctx, bucket, ev)
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
			l: setup.NewLog("lambda:create-repo-metadata"),
			s3: &storage.S3{
				Session: s,
			},
		}

		lambda.Start((&f).HandleRequest)
		return nil
	})
}
