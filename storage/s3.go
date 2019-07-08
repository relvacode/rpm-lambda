package storage

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"git.illumina.com/relvacode/rpm-lambda/yum"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"io"
)

func simpleConcurrentError(f func() error) chan error {
	err := make(chan error, 1)
	go func() {
		err <- f()
		close(err)
	}()
	return err
}

type S3 struct {
	*session.Session
}

func (storage *S3) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := s3.New(storage).DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (storage *S3) DownloadObject(ctx context.Context, bucket, key string) (bool, io.ReadCloser, error) {
	o, err := s3.New(storage).GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		if ex, ok := err.(awserr.Error); ok {
			if ex.Code() == s3.ErrCodeNoSuchKey {
				return false, nil, nil
			}
		}
		return false, nil, errors.Wrap(err, "download object")
	}
	return true, o.Body, nil
}

func (storage *S3) DownloadXMLObject(ctx context.Context, data interface{}, bucket, key string) (bool, error) {
	found, r, err := storage.DownloadObject(ctx, bucket, key)
	if err != nil {
		return false, errors.Wrap(err, "download XML object")
	}
	if !found {
		return false, nil
	}

	defer r.Close()
	return true, xml.NewDecoder(r).Decode(data)
}

func (storage *S3) uploader() *s3manager.Uploader {
	uploader := s3manager.NewUploader(storage)
	uploader.Concurrency = 1
	uploader.PartSize = 5 << 20
	return uploader
}

func (storage *S3) UploadObject(ctx context.Context, r io.Reader, bucket, key, content string) error {
	_, err := storage.uploader().UploadWithContext(ctx, &s3manager.UploadInput{
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		Bucket:      aws.String(bucket),
		ContentType: aws.String(content),
		Key:         aws.String(key),
		Body:        r,
	})
	return err
}

func (storage *S3) UploadXMLObject(ctx context.Context, data interface{}, bucket, key string) error {
	var (
		pr, pw = io.Pipe()
		e      = xml.NewEncoder(pw)
	)

	e.Indent("", "  ")

	errs := simpleConcurrentError(func() error {
		err := e.Encode(data)
		_ = pw.CloseWithError(err)
		return err
	})

	err := storage.UploadObject(ctx, pr, bucket, key, "text/xml")
	_ = pr.Close()
	if err != nil {
		return errors.Wrapf(err, "failed to upload %q into %q", key, bucket)
	}

	err = <-errs
	if err != nil {
		return err
	}

	return nil
}

func (storage *S3) DownloadCompressedXMLObject(ctx context.Context, data interface{}, bucket, key string) (bool, error) {
	found, r, err := storage.DownloadObject(ctx, bucket, key)
	if err != nil {
		return false, errors.Wrap(err, "download compressed XML object")
	}
	if !found {
		return false, nil
	}

	defer r.Close()
	g, err := gzip.NewReader(r)
	if err != nil {
		return false, err
	}

	_ = g.Close()
	return true, xml.NewDecoder(g).Decode(data)
}

func (storage *S3) UploadCompressedXMLObject(ctx context.Context, data interface{}, bucket, key string) (*XMLObject, error) {
	var (
		pr, pw        = io.Pipe()
		shaContent    = yum.SHA256() // SHA256 of raw content
		shaCompressed = yum.SHA256() // SHA256 of compressed data
		g             = gzip.NewWriter(io.MultiWriter(pw, shaCompressed))
		w             = io.MultiWriter(g, shaContent)
		e             = xml.NewEncoder(w)
	)

	e.Indent("", "  ")

	errs := simpleConcurrentError(func() (err error) {
		defer pw.CloseWithError(err)

		err = e.Encode(data)
		if err != nil {
			return
		}

		err = g.Close()
		if err != nil {
			return
		}

		return
	})

	err := storage.UploadObject(ctx, pr, bucket, key, "application/x-gzip")
	_ = pr.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upload %q into %q", key, bucket)
	}

	err = <-errs
	if err != nil {
		return nil, err
	}

	return &XMLObject{
		Key:             key,
		ContentChecksum: shaContent.Sum(),
		ObjectChecksum:  shaCompressed.Sum(),
	}, nil
}
