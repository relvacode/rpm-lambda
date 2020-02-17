# rpm-lambda

AWS Lambda functions for RPM code signing and YUM repository generation.

## Overview

### sign-package

The `sign-package` lambda allows you to automatically sign RPM files being uploaded to an S3 bucket.

It downloads the RPM file at the patch received by the event, signs it with a GPG private key loaded from AWS Secrets Manage, and uploads the signed RPM to the same path on a target S3 bucket.

### create-repo-metadata

TBD

### sign-repo-metadata

## Compile

```
docker run -t --rm -w /src/ -v $PWD:/src/ --entrypoint bash golang:1.13-buster -c 'go build -o /src/sign-package ./lambdas/sign-package'
docker run -t --rm -w /src/ -v $PWD:/src/ --entrypoint bash golang:1.13-buster -c 'go build -o /src/create-repo-metadata ./lambdas/create-repo-metadata'
docker run -t --rm -w /src/ -v $PWD:/src/ --entrypoint bash golang:1.13-buster -c 'go build -o /src/sign-repo-metadata ./lambdas/sign-repo-metadata'
```

## Install

### sign-package

- Create the needed aws secrets. You will need both a gpg private key and the passphrase protecting it (you can use whatever names you want for them). `sign-package` expects the secrets to be stored in a binary form, so you have to use the CLI for this (the web console doesn't supports binary secrets yet):
```
aws secretsmanager create-secret --name gpg_key --secret-binary file:///path/to/gpg_private_key
aws secretsmanager create-secret --name gpg_passphrase --secret-binary file:///path/to/passphrase
```
- Create an IAM role for the lambda function with the following policies:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowCreatingLogGroup",
            "Effect": "Allow",
            "Action": "logs:CreateLogGroup",
            "Resource": "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:*"
        },
        {
            "Sid": "AllowCreatingLogStreamsAndEvents",
            "Sid": "ManageS3IncomingBucket",
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": [
                "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:log-group:/aws/lambda/rpm-sign:*"
            ]
        }
    ]
}
```
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "ManageS3IncomingBucket",
            "Effect": "Allow",
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::${S3_IN_BUCKET}"
        },
        {
            "Sid": "ManageS3TargetBucket",
            "Effect": "Allow",
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::${S3_TARGET_BUCKET}"
        },
        {
            "Sid": "ManageS3IncomingBucketObjects",
            "Effect": "Allow",
            "Action": [
                "s3:DeleteObjectTagging",
                "s3:DeleteObjectVersion",
                "s3:GetObjectVersionTagging",
                "s3:PutObjectVersionTagging",
                "s3:DeleteObjectVersionTagging",
                "s3:ListMultipartUploadParts",
                "s3:PutObject",
                "s3:GetObjectAcl",
                "s3:GetObject",
                "s3:AbortMultipartUpload",
                "s3:PutObjectVersionAcl",
                "s3:GetObjectVersionAcl",
                "s3:GetObjectTagging",
                "s3:PutObjectTagging",
                "s3:DeleteObject",
                "s3:PutObjectAcl",
                "s3:GetObjectVersion"
            ],
            "Resource": "arn:aws:s3:::${S3_IN_BUCKET}/*"
        },
        {
            "Sid": "ManageS3TargetBucketObjects",
            "Effect": "Allow",
            "Action": [
                "s3:DeleteObjectTagging",
                "s3:DeleteObjectVersion",
                "s3:GetObjectVersionTagging",
                "s3:PutObjectVersionTagging",
                "s3:DeleteObjectVersionTagging",
                "s3:ListMultipartUploadParts",
                "s3:PutObject",
                "s3:GetObjectAcl",
                "s3:GetObject",
                "s3:AbortMultipartUpload",
                "s3:PutObjectVersionAcl",
                "s3:GetObjectVersionAcl",
                "s3:GetObjectTagging",
                "s3:PutObjectTagging",
                "s3:DeleteObject",
                "s3:PutObjectAcl",
                "s3:GetObjectVersion"
            ],
            "Resource": "arn:aws:s3:::${S3_TARGET_BUCKET}/*"
        }
    ]
}
```
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "ReadSecrets",
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetRandomPassword",
                "secretsmanager:GetResourcePolicy",
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret",
                "secretsmanager:ListSecrets",
                "secretsmanager:ListSecretVersionIds"
            ],
            "Resource": [
                "arn:aws:secretsmanager:${AWS_REGION}:${AWS_ACCOUNT_ID}:secret:${GPG_KEY_ARN_NAME}",
                "arn:aws:secretsmanager:${AWS_REGION}:${AWS_ACCOUNT_ID}:secret:a${GPG_PASSPHRASE_ARN_NAME}"
            ]
        }
    ]
}
```
- Compile the `sign-package` binary (see above) and zip it.
- Create the lambda with the `Go 1.x` runtime, upload the zip archive created above and set the following environment variables:
  - `LAMBDA_SECRET_GPG_KEY`: The name you used for the gpg private key aws secret, e.g. `gpg_key` in the example above
  - `LAMBDA_SECRET_GPG_PASSPHRASE`: The name you used for the gpg passphrase aws secret, e.g. `gpg_passphrase` in the example above
  - `LAMBDA_S3_TARGET`: the name of your target bucket

### create-repo-metadata

TBD

### sign-repo-metadata

TBD

- `LAMBDA_SECRET_GPG_KEY`: The name you used for the gpg private key aws secret, e.g. `gpg_key` in the example above
- `LAMBDA_SECRET_GPG_PASSPHRASE`: The name you used for the gpg passphrase aws secret, e.g. `gpg_passphrase` in the example above
