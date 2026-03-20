# S3-to-GCS Management API Proxy

A lightweight Go proxy that accepts native AWS S3 SDK API calls for bucket management operations and translates them to Google Cloud Storage (GCS) API calls. This enables customers to use standard S3 SDK tooling to manage GCS buckets for versioning, logging, CORS, and tagging.

## Why This Project

Google Cloud Storage provides S3-compatible XML APIs for object operations, but the bucket management APIs (versioning, logging, CORS, tagging) are not fully compatible with their AWS S3 counterparts. This proxy bridges that gap by:

- Accepting standard S3 XML API requests from any S3 SDK client
- Converting them to GCS JSON SDK calls
- Returning S3-compatible XML responses

This allows teams already using S3 SDKs or tooling (such as AWS CLI, Terraform AWS provider, or application code) to manage GCS bucket configurations without rewriting their code.

## Supported APIs

| Operation | Methods | S3 Query Parameter |
|-----------|---------|-------------------|
| Bucket Versioning | GET, PUT | `?versioning` |
| Bucket CORS | GET, PUT, DELETE | `?cors` |
| Bucket Logging | GET, PUT | `?logging` |
| Bucket Tagging | GET, PUT, DELETE | `?tagging` |

### S3 to GCS Field Mapping

| S3 Field | GCS Equivalent | Notes |
|----------|---------------|-------|
| Versioning `Status` (Enabled/Suspended) | `VersioningEnabled` (true/false) | |
| Versioning `MfaDelete` | Not supported | Returns `InvalidArgument` error |
| CORS `AllowedOrigin` | `Origins` | |
| CORS `AllowedMethod` | `Methods` | |
| CORS `ExposeHeader` | `ResponseHeaders` | |
| CORS `MaxAgeSeconds` | `MaxAge` | |
| CORS `AllowedHeader` | No direct equivalent | Silently accepted (GCS allows all request headers) |
| CORS `ID` | No equivalent | Silently dropped |
| Logging `TargetBucket` | `LogBucket` | |
| Logging `TargetPrefix` | `LogObjectPrefix` | |
| Logging `TargetGrants` | Not supported | Returns `InvalidArgument` error |
| Tagging `Tag` (Key/Value) | Labels (key/value) | Keys are automatically lowercased |

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │              S3-GCS Proxy               │
                    │                                         │
S3 SDK Client ─────▶  Middleware (Recovery, Logging, BodyLimit,│
(XML over HTTP)     │             Signature Bypass)           │
                    │         │                               │
                    │         ▼                               │
                    │  Router (bucket extraction +            │
                    │          query param dispatch)          │
                    │         │                               │
                    │         ▼                               │
                    │  Handler (XML parse/generate)           │
                    │         │                               │
                    │         ▼                               │
                    │  Converter (S3 model ↔ GCS model)       │
                    │         │                               │
                    │         ▼                               │
                    │  GCS Client (cloud.google.com/go/storage)│
                    └─────────┬───────────────────────────────┘
                              │
                              ▼
                    Google Cloud Storage API
```

The proxy supports both S3 addressing styles:
- **Path style**: `http://proxy:8080/my-bucket?versioning`
- **Virtual-hosted style**: `http://my-bucket.proxy:8080/?versioning`

S3 v4 signatures are detected but intentionally not validated (per design).

## Prerequisites

- **Go 1.25+** (for building from source)
- **Google Cloud credentials**: The proxy uses [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials). Set up using one of:
  - `gcloud auth application-default login` (for local development)
  - Service account key file via `GOOGLE_APPLICATION_CREDENTIALS` env var
  - Workload Identity (for GKE deployments)
  - Attached service account (for GCE/Cloud Run)
- **GCS bucket**: The target bucket must already exist in your GCP project
- **IAM permissions**: The credentials must have `storage.buckets.get` and `storage.buckets.update` on the target buckets

## Build

```bash
git clone https://github.com/nonokangwei/s3management.git
cd s3management
go build -o s3-gcs-proxy .
```

## Deployment

### Option 1: Run Directly

```bash
# Set required environment variables
export GCS_PROJECT_ID=my-gcp-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Start the proxy
./s3-gcs-proxy --addr :8080
```

### Option 2: Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o s3-gcs-proxy .

FROM gcr.io/distroless/static
COPY --from=builder /app/s3-gcs-proxy /s3-gcs-proxy
ENTRYPOINT ["/s3-gcs-proxy"]
```

Build and run:

```bash
docker build -t s3-gcs-proxy .
docker run -p 8080:8080 \
  -e GCS_PROJECT_ID=my-gcp-project \
  -v /path/to/service-account.json:/creds.json \
  -e GOOGLE_APPLICATION_CREDENTIALS=/creds.json \
  s3-gcs-proxy
```

### Option 3: Deploy on GKE / Cloud Run

For GKE or Cloud Run, use Workload Identity or an attached service account so no key file is needed:

```bash
# Cloud Run example
gcloud run deploy s3-gcs-proxy \
  --image gcr.io/my-project/s3-gcs-proxy \
  --set-env-vars GCS_PROJECT_ID=my-gcp-project \
  --port 8080 \
  --allow-unauthenticated
```

## Configuration

All settings can be configured via environment variables or command-line flags. Flags take precedence.

| Environment Variable | Flag | Default | Description |
|---------------------|------|---------|-------------|
| `GCS_PROJECT_ID` | `--project` | (required) | Google Cloud project ID |
| `LISTEN_ADDR` | `--addr` | `:8080` | Listen address (host:port) |
| `LOG_LEVEL` | `--log-level` | `info` | Log level: debug, info, warn, error |
| `GOOGLE_APPLICATION_CREDENTIALS` | — | ADC | Path to GCP service account key |
| `READ_TIMEOUT` | — | `30s` | Max duration for reading requests |
| `WRITE_TIMEOUT` | — | `30s` | Max duration for writing responses |
| `IDLE_TIMEOUT` | — | `120s` | Keep-alive connection idle timeout |
| `SHUTDOWN_TIMEOUT` | — | `15s` | Graceful shutdown timeout |
| `GCS_REQUEST_TIMEOUT` | — | `30s` | Timeout for each GCS API call |
| `MAX_REQUEST_BODY_KB` | — | `256` | Max request body size in KB |

## Usage

### Health Check

```bash
curl http://localhost:8080/healthz
# ok
```

### AWS CLI

Configure the AWS CLI to point at the proxy:

```bash
# Set bucket versioning
aws s3api put-bucket-versioning \
  --bucket my-gcs-bucket \
  --versioning-configuration Status=Enabled \
  --endpoint-url http://localhost:8080

# Get bucket versioning
aws s3api get-bucket-versioning \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080

# Set bucket CORS
aws s3api put-bucket-cors \
  --bucket my-gcs-bucket \
  --cors-configuration '{"CORSRules":[{"AllowedOrigins":["*"],"AllowedMethods":["GET","PUT"],"MaxAgeSeconds":3600}]}' \
  --endpoint-url http://localhost:8080

# Get bucket CORS
aws s3api get-bucket-cors \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080

# Delete bucket CORS
aws s3api delete-bucket-cors \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080

# Set bucket tagging
aws s3api put-bucket-tagging \
  --bucket my-gcs-bucket \
  --tagging 'TagSet=[{Key=env,Value=prod},{Key=project,Value=myapp}]' \
  --endpoint-url http://localhost:8080

# Get bucket tagging
aws s3api get-bucket-tagging \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080

# Delete bucket tagging
aws s3api delete-bucket-tagging \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080

# Set bucket logging
aws s3api put-bucket-logging \
  --bucket my-gcs-bucket \
  --bucket-logging-status '{"LoggingEnabled":{"TargetBucket":"my-log-bucket","TargetPrefix":"logs/"}}' \
  --endpoint-url http://localhost:8080

# Get bucket logging
aws s3api get-bucket-logging \
  --bucket my-gcs-bucket \
  --endpoint-url http://localhost:8080
```

### AWS SDK for Go v2

```go
package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func main() {
    client := s3.New(s3.Options{
        BaseEndpoint: aws.String("http://localhost:8080"),
        Region:       "us-east-1",
        Credentials:  credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
        UsePathStyle: true,
    })

    // Enable versioning
    _, err := client.PutBucketVersioning(context.Background(), &s3.PutBucketVersioningInput{
        Bucket: aws.String("my-gcs-bucket"),
        VersioningConfiguration: &types.VersioningConfiguration{
            Status: types.BucketVersioningStatusEnabled,
        },
    })
    if err != nil {
        panic(err)
    }

    // Get versioning
    result, _ := client.GetBucketVersioning(context.Background(), &s3.GetBucketVersioningInput{
        Bucket: aws.String("my-gcs-bucket"),
    })
    fmt.Printf("Versioning status: %s\n", result.Status)
}
```

### Python (boto3)

```python
import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:8080',
    aws_access_key_id='dummy',
    aws_secret_access_key='dummy',
    region_name='us-east-1',
)

# Enable versioning
s3.put_bucket_versioning(
    Bucket='my-gcs-bucket',
    VersioningConfiguration={'Status': 'Enabled'},
)

# Get versioning
response = s3.get_bucket_versioning(Bucket='my-gcs-bucket')
print(f"Versioning: {response.get('Status')}")
```

## Error Handling

The proxy returns S3-compatible XML error responses:

| Scenario | S3 Error Code | HTTP Status |
|----------|--------------|-------------|
| Bucket not found | `NoSuchBucket` | 404 |
| Unsupported field (MfaDelete, TargetGrants) | `InvalidArgument` | 400 |
| Malformed request XML | `MalformedXML` | 400 |
| GCS permission denied | `AccessDenied` | 403 |
| GCS rate limit exceeded | `SlowDown` | 503 |
| GCS request timeout | `RequestTimeout` | 408 |
| GCS internal error | `InternalError` | 500 |

## Testing

```bash
# Run all unit and integration tests
go test ./...

# Run with verbose output
go test -v ./...

# Run E2E tests against a real GCS bucket (requires GCP credentials)
go test -v -run TestE2E .
```

## Project Structure

```
s3management/
├── main.go                  # Entry point, server startup, graceful shutdown
├── config/config.go         # Configuration (env vars + flags)
├── model/
│   ├── s3xml.go             # S3 XML request/response structs
│   └── errors.go            # S3-compatible error responses
├── gcs/client.go            # GCS SDK wrapper + BucketOperator interface
├── converter/               # S3 XML <-> GCS field conversion
│   ├── versioning.go
│   ├── cors.go
│   ├── logging.go
│   └── tagging.go
├── handler/                 # HTTP handlers for each API
│   ├── versioning.go
│   ├── cors.go
│   ├── logging.go
│   ├── tagging.go
│   └── common.go
├── server/router.go         # HTTP routing + bucket name extraction
├── middleware/
│   ├── signature.go         # S3 v4 signature bypass
│   ├── recovery.go          # Panic recovery
│   ├── requestlog.go        # Request logging with latency
│   └── bodylimit.go         # Request body size limit
└── e2e_test.go              # End-to-end tests with real S3 SDK
```

## License

Apache 2.0
