package delegation_backend

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3OptionsFromEnv applies environment-driven S3 client options for use with
// S3-compatible servers like MinIO or LocalStack. Production AWS S3 requires
// neither variable.
//
// AWS_ENDPOINT_URL_S3 overrides the S3 endpoint URL. Native support for this
// env var was added to aws-sdk-go-v2/config v1.27 (Feb 2024); this backend is
// pinned to an older version, so we read it explicitly here.
//
// AWS_S3_FORCE_PATH_STYLE=1 enables path-style addressing
// (https://endpoint/bucket/key instead of https://bucket.endpoint/key).
// MinIO and LocalStack don't host buckets as DNS subdomains, so this is
// required when pointing the backend at them.
func S3OptionsFromEnv(o *s3.Options) {
	if endpoint := os.Getenv("AWS_ENDPOINT_URL_S3"); endpoint != "" {
		o.BaseEndpoint = aws.String(endpoint)
	}
	if os.Getenv("AWS_S3_FORCE_PATH_STYLE") == "1" {
		o.UsePathStyle = true
	}
}
