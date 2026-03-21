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

### Compatibility Matrix (S3 ↔︎ GCS)

| API | Support | Notes / Known Differences |
| --- | --- | --- |
| Versioning | Supported | `MfaDelete` is rejected with `InvalidArgument`; maps to GCS `VersioningEnabled` boolean. |
| CORS | Supported | `AllowedHeader` accepted but ignored (GCS allows all request headers); `ID` dropped. |
| Logging | Partial | `TargetGrants` not supported; same bucket can be used as log target. |
| Tagging | Supported | Keys lowercased to satisfy GCS label constraints; deleting tags removes all labels. |
| Error model | Supported | Centralized GCS→S3 mapping; responses include `RequestId` and S3 XML envelope. |

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

### Contract Docs

- OpenAPI: `docs/openapi.yaml` documents every operation with schemas for XML payloads and S3 errors.
- Metrics: `/metrics` exposes Prometheus metrics (`s3_proxy_request_latency_seconds`, `s3_proxy_upstream_errors_total`).
- Request correlation: every response includes `X-Request-ID`; S3 error payloads echo this as `RequestId`.

#### cURL Examples (real wire paths)

```bash
# Versioning
curl -H "X-Request-ID: demo-1" -X GET  http://localhost:8080/my-bucket?versioning
curl -H "Content-Type: application/xml" -d '<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>' \
  -X PUT http://localhost:8080/my-bucket?versioning

# CORS
curl -X GET http://localhost:8080/my-bucket?cors
curl -H "Content-Type: application/xml" -d '<CORSConfiguration><CORSRule><AllowedOrigin>*</AllowedOrigin><AllowedMethod>GET</AllowedMethod></CORSRule></CORSConfiguration>' \
  -X PUT http://localhost:8080/my-bucket?cors
curl -X DELETE http://localhost:8080/my-bucket?cors

# Logging
curl -X GET http://localhost:8080/my-bucket?logging
curl -H "Content-Type: application/xml" -d '<BucketLoggingStatus><LoggingEnabled><TargetBucket>logs</TargetBucket><TargetPrefix>prefix/</TargetPrefix></LoggingEnabled></BucketLoggingStatus>' \
  -X PUT http://localhost:8080/my-bucket?logging

# Tagging
curl -X GET http://localhost:8080/my-bucket?tagging
curl -H "Content-Type: application/xml" -d '<Tagging><TagSet><Tag><Key>env</Key><Value>prod</Value></Tag></TagSet></Tagging>' \
  -X PUT http://localhost:8080/my-bucket?tagging
curl -X DELETE http://localhost:8080/my-bucket?tagging
```

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

## Operations, Observability, and Config Hardening

- **Request IDs**: Every response includes `X-Request-ID`; errors echo this value inside the XML body for correlation.
- **Metrics**: Prometheus scrape endpoint at `/metrics` with latency histograms and upstream GCS error counters labeled by operation/bucket.
- **Timeouts**: Per-request GCS calls are wrapped in `GCS_REQUEST_TIMEOUT` (default 30s); startup enforces a non-zero timeout and fails fast on invalid env vars.
- **Config validation**: All env vars are strictly parsed at startup; missing or malformed values abort the process with a clear message.
- **Health**: `/healthz` returns `200 OK` when the process is alive.

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

## CI and Security Notes

- CI runs `go test ./...`, `go test -race ./...`, `go vet ./...`, and `govulncheck ./...` on pull requests.
- Security/operations: Dependabot opened **Bump google.golang.org/grpc from 1.79.2 to 1.79.3** (includes upstream security fixes). Merge once CI passes.

### Option 3: Deploy on GKE

Use [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) so the proxy can access GCS without managing key files.

**1. Set up Workload Identity**

```bash
# Create a GCP service account
gcloud iam service-accounts create s3-gcs-proxy \
  --display-name="S3-GCS Proxy"

# Grant GCS permissions
gcloud projects add-iam-policy-binding my-gcp-project \
  --member="serviceAccount:s3-gcs-proxy@my-gcp-project.iam.gserviceaccount.com" \
  --role="roles/storage.admin"

# Allow the Kubernetes service account to impersonate the GCP service account
gcloud iam service-accounts add-iam-policy-binding \
  s3-gcs-proxy@my-gcp-project.iam.gserviceaccount.com \
  --role="roles/iam.workloadIdentityUser" \
  --member="serviceAccount:my-gcp-project.svc.id.goog[s3-gcs-proxy/s3-gcs-proxy]"
```

**2. Build and push the image**

```bash
docker build -t gcr.io/my-gcp-project/s3-gcs-proxy .
docker push gcr.io/my-gcp-project/s3-gcs-proxy
```

**3. Apply Kubernetes manifests**

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: s3-gcs-proxy
---
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: s3-gcs-proxy
  namespace: s3-gcs-proxy
  annotations:
    iam.gke.io/gcp-service-account: s3-gcs-proxy@my-gcp-project.iam.gserviceaccount.com
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: s3-gcs-proxy
  namespace: s3-gcs-proxy
  labels:
    app: s3-gcs-proxy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: s3-gcs-proxy
  template:
    metadata:
      labels:
        app: s3-gcs-proxy
    spec:
      serviceAccountName: s3-gcs-proxy
      containers:
        - name: s3-gcs-proxy
          image: gcr.io/my-gcp-project/s3-gcs-proxy:latest
          args: ["--project", "my-gcp-project"]
          ports:
            - containerPort: 8080
              protocol: TCP
          env:
            - name: LISTEN_ADDR
              value: ":8080"
            - name: GCS_REQUEST_TIMEOUT
              value: "30s"
            - name: MAX_REQUEST_BODY_KB
              value: "256"
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 3
            periodSeconds: 5
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 256Mi
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: s3-gcs-proxy
  namespace: s3-gcs-proxy
spec:
  selector:
    app: s3-gcs-proxy
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
  type: ClusterIP
```

Deploy:

```bash
kubectl apply -f namespace.yaml
kubectl apply -f serviceaccount.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

To expose externally, add an Ingress or change Service type to `LoadBalancer`:

```yaml
# service-lb.yaml (alternative)
apiVersion: v1
kind: Service
metadata:
  name: s3-gcs-proxy-lb
  namespace: s3-gcs-proxy
spec:
  selector:
    app: s3-gcs-proxy
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
  type: LoadBalancer
```

### Option 4: Deploy on Cloud Run

```bash
gcloud run deploy s3-gcs-proxy \
  --image gcr.io/my-gcp-project/s3-gcs-proxy \
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

## Test Results

**Overall: 45/45 PASSED**

### Unit Tests — XML Parsing/Generation (`model`)

| Test | Description | Result |
|------|-------------|--------|
| VersioningConfigurationXML/enabled | Parse `<Status>Enabled</Status>` and round-trip | ✅ |
| VersioningConfigurationXML/suspended | Parse `<Status>Suspended</Status>` and round-trip | ✅ |
| VersioningConfigurationXML/mfa_delete | Parse MfaDelete field | ✅ |
| VersioningConfigurationXML/empty | Parse empty VersioningConfiguration | ✅ |
| CORSConfigurationXML | Parse multi-origin, multi-method CORS rule with all fields | ✅ |
| BucketLoggingStatusXML/enabled | Parse LoggingEnabled with TargetBucket/TargetPrefix | ✅ |
| BucketLoggingStatusXML/disabled | Parse empty BucketLoggingStatus | ✅ |
| TaggingXML | Parse TagSet with multiple tags and round-trip | ✅ |
| VersioningConfigurationWithNamespace | Parse XML with S3 namespace | ✅ |
| BucketLoggingWithTargetGrants | Parse logging XML containing TargetGrants | ✅ |

### Unit Tests — S3/GCS Conversion (`converter`)

| Test | Description | Result |
|------|-------------|--------|
| VersioningToGCS_Enabled | S3 `Enabled` → GCS `true` | ✅ |
| VersioningToGCS_Suspended | S3 `Suspended` → GCS `false` | ✅ |
| VersioningToGCS_MfaDeleteError | MfaDelete → rejected with error | ✅ |
| VersioningToGCS_InvalidStatus | Invalid status string → error | ✅ |
| VersioningFromGCS_Enabled | GCS `true` → S3 `Enabled` | ✅ |
| VersioningFromGCS_Disabled | GCS `false` → S3 `Suspended` | ✅ |
| CORSToGCS | S3 CORS → GCS CORS (Origins, Methods, ResponseHeaders, MaxAge) | ✅ |
| CORSFromGCS | GCS CORS → S3 CORS (reverse mapping) | ✅ |
| CORSToGCS_EmptyRules | Empty S3 CORS → empty GCS CORS slice | ✅ |
| LoggingToGCS | S3 LoggingEnabled → GCS BucketLogging | ✅ |
| LoggingToGCS_DisableLogging | Nil LoggingEnabled → empty GCS BucketLogging | ✅ |
| LoggingToGCS_TargetGrantsError | TargetGrants → rejected with error | ✅ |
| LoggingFromGCS | GCS BucketLogging → S3 LoggingEnabled | ✅ |
| LoggingFromGCS_NoLogging | Nil GCS Logging → nil LoggingEnabled | ✅ |
| TaggingToGCS | S3 Tags → GCS labels (with key lowercasing) | ✅ |
| TaggingDeleteToGCS | Build delete-all-labels update | ✅ |
| TaggingFromGCS | GCS labels → S3 Tags | ✅ |
| TaggingFromGCS_NoLabels | Nil labels → empty TagSet | ✅ |

### Unit Tests — Router (`server`)

| Test | Description | Result |
|------|-------------|--------|
| ExtractBucketName_PathStyle/simple | `/mybucket` on `localhost:8080` | ✅ |
| ExtractBucketName_PathStyle/trailing_slash | `/mybucket/` on `localhost:8080` | ✅ |
| ExtractBucketName_PathStyle/query | `/mybucket` on `s3.amazonaws.com` | ✅ |
| ExtractBucketName_VirtualHosted | `mybucket.s3.amazonaws.com` | ✅ |
| ExtractBucketName_Empty | `/` on `localhost:8080` → empty | ✅ |

### Integration Tests — Handlers with Mock GCS (`handler`)

| Test | Description | Result |
|------|-------------|--------|
| GetVersioning | GET /?versioning → 200, XML with `Enabled` | ✅ |
| PutVersioning_Enabled | PUT /?versioning with Enabled → 200 | ✅ |
| PutVersioning_MfaDeleteError | PUT with MfaDelete → 400 InvalidArgument | ✅ |
| PutVersioning_MalformedXML | PUT with invalid XML → 400 MalformedXML | ✅ |
| GetVersioning_BucketNotFound | GET nonexistent bucket → 404 NoSuchBucket | ✅ |
| GetCORS | GET /?cors → 200, XML with CORS rules | ✅ |
| PutCORS | PUT /?cors → 200 | ✅ |
| DeleteCORS | DELETE /?cors → 204 | ✅ |
| GetLogging | GET /?logging → 200, XML with LoggingEnabled | ✅ |
| PutLogging | PUT /?logging → 200 | ✅ |
| PutLogging_TargetGrantsError | PUT with TargetGrants → 400 InvalidArgument | ✅ |
| PutLogging_DisableLogging | PUT with empty BucketLoggingStatus → 200 | ✅ |
| GetTagging | GET /?tagging → 200, XML with tags | ✅ |
| PutTagging | PUT /?tagging → 200 | ✅ |
| DeleteTagging | DELETE /?tagging → 204 | ✅ |
| DeleteTagging_NoLabels | DELETE with no existing labels → 204 | ✅ |

### End-to-End Tests — Real GCS Bucket with AWS S3 SDK v2

E2E tests run against a real GCS bucket using the AWS S3 SDK v2 Go client through the proxy.

| Test | Steps | Result |
|------|-------|--------|
| E2E_Versioning | Enable → GET (Enabled) → Suspend → GET (Suspended) | ✅ |
| E2E_CORS | PUT 2 origins/methods → GET (verify) → DELETE → GET (empty) | ✅ |
| E2E_Tagging | PUT 2 tags → GET (verify values) → DELETE → GET (empty) | ✅ |
| E2E_Logging | PUT enable (bucket + prefix) → GET (verify) → PUT disable → GET (nil) | ✅ |
| E2E_Versioning_MfaDeleteError | PUT with MfaDelete=Enabled → 400 InvalidArgument | ✅ |

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
