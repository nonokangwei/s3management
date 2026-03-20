# S3-to-GCS Proxy Implementation Plan

## Context

Google Cloud's native XML APIs are not fully compatible with AWS S3 XML API for bucket versioning, logging, CORS, and tagging operations. This project builds a Go proxy that accepts native S3 SDK API calls (XML format) and converts them to GCS JSON SDK calls, enabling customers to use standard S3 SDK tooling against a GCS backend for these specific management APIs.

This is a greenfield project — only `design.md` exists today.

---

## Project Structure

```
s3management/
├── main.go                      # Entry point, server startup, graceful shutdown
├── go.mod / go.sum
├── config/
│   └── config.go                # Config struct (listen addr, GCS project ID, log level)
├── model/
│   ├── s3xml.go                 # S3 XML request/response structs (all 4 features)
│   └── errors.go                # S3-compatible error XML struct + WriteS3Error helper
├── gcs/
│   └── client.go                # BucketOperator interface + GCS SDK wrapper
├── converter/
│   ├── versioning.go            # S3 versioning XML <-> GCS VersioningEnabled
│   ├── cors.go                  # S3 CORS XML <-> GCS CORS attrs
│   ├── logging.go               # S3 logging XML <-> GCS Logging attrs
│   └── tagging.go               # S3 tagging XML <-> GCS Labels map
├── handler/
│   ├── versioning.go            # PUT/GET /?versioning
│   ├── cors.go                  # PUT/GET/DELETE /?cors
│   ├── logging.go               # PUT/GET/DELETE /?logging
│   ├── tagging.go               # PUT/GET/DELETE /?tagging
│   └── common.go                # Shared helpers (XML response writer, GCS error mapping)
├── server/
│   └── router.go                # HTTP routing (bucket name extraction + query param dispatch)
├── middleware/
│   └── signature.go             # S3 v4 signature bypass (detect and pass through)
└── tests/
    ├── model/s3xml_test.go      # Unit: XML parsing/generation round-trips
    ├── converter/converter_test.go  # Unit: S3<->GCS field conversion
    ├── handler/handler_test.go  # Integration: handlers with mock GCS client
    ├── server/router_test.go    # Unit: bucket name extraction and routing
    └── e2e_test.go              # E2E: real AWS S3 SDK v2 client against proxy
```

## Dependencies

- `cloud.google.com/go/storage` — GCS Go SDK
- `google.golang.org/api/option` — GCS client options
- Standard library: `net/http`, `encoding/xml`, `log`, `os`, `flag`, `io`, `context`, `os/signal`
- Test only: `github.com/aws/aws-sdk-go-v2/service/s3` (for E2E tests)

No third-party HTTP router — `net/http` suffices for query-param-based dispatch.

## API Endpoints (10 total)

| Query Param   | Method | Handler                | S3 XML Root              |
|---------------|--------|------------------------|--------------------------|
| `?versioning` | GET    | `GetVersioning`        | `VersioningConfiguration`|
| `?versioning` | PUT    | `PutVersioning`        | `VersioningConfiguration`|
| `?cors`       | GET    | `GetCORS`              | `CORSConfiguration`      |
| `?cors`       | PUT    | `PutCORS`              | `CORSConfiguration`      |
| `?cors`       | DELETE | `DeleteCORS`           | —                        |
| `?logging`    | GET    | `GetLogging`           | `BucketLoggingStatus`    |
| `?logging`    | PUT    | `PutLogging`           | `BucketLoggingStatus`    |
| `?tagging`    | GET    | `GetTagging`           | `Tagging`                |
| `?tagging`    | PUT    | `PutTagging`           | `Tagging`                |
| `?tagging`    | DELETE | `DeleteTagging`        | —                        |

## S3 to GCS Field Mapping

### Versioning
- S3 `Status: "Enabled"` → GCS `VersioningEnabled: true`
- S3 `Status: "Suspended"` → GCS `VersioningEnabled: false`
- S3 `MfaDelete` → **Not supported** — return `InvalidArgument` error

### CORS
- S3 `AllowedOrigin` → GCS `Origins`
- S3 `AllowedMethod` → GCS `Methods`
- S3 `ExposeHeader` → GCS `ResponseHeaders`
- S3 `MaxAgeSeconds` → GCS `MaxAge` (as `time.Duration`)
- S3 `AllowedHeader` → No direct GCS equivalent (silently accepted, documented in comments)
- S3 `ID` → No GCS equivalent (silently dropped)
- DELETE: Update with empty CORS slice

### Logging
- S3 `TargetBucket` → GCS `LogBucket`
- S3 `TargetPrefix` → GCS `LogObjectPrefix`
- S3 `TargetGrants` → **Not supported** — return `InvalidArgument` error
- Disable logging: PUT with empty `BucketLoggingStatus` (no `LoggingEnabled` element) — no DELETE API exists in S3

### Tagging → Labels
- S3 `Tag{Key, Value}` → GCS `SetLabel(lowercase(key), value)`
- GCS requires lowercase label keys — converter lowercases automatically
- DELETE: Fetch current labels, call `DeleteLabel` for each key

## Key Design Decisions

1. **BucketOperator interface** in `gcs/client.go` — enables mock-based testing of handlers without a real GCS backend
2. **Separate converter layer** — keeps conversion logic independently testable, handlers stay thin
3. **Bucket name extraction** supports both virtual-hosted style (`mybucket.s3.amazonaws.com`) and path style (`/mybucket`)
4. **S3 v4 signature** detected but never validated (per design doc)
5. **Unsupported fields** (MfaDelete, TargetGrants) return S3-compatible error XML immediately

## Error Handling

S3-compatible XML error responses:
| Scenario                 | S3 Code          | HTTP Status |
|--------------------------|------------------|-------------|
| Bucket not found         | `NoSuchBucket`   | 404         |
| Unsupported field        | `InvalidArgument`| 400         |
| Malformed XML            | `MalformedXML`   | 400         |
| GCS permission denied    | `AccessDenied`   | 403         |
| GCS internal error       | `InternalError`  | 500         |

## Implementation Order

| Step | Files | What |
|------|-------|------|
| 1 | `go.mod` | Init module, add GCS dependency |
| 2 | `config/config.go` | Config from env vars + flags |
| 3 | `model/s3xml.go` | All S3 XML structs (4 features) |
| 4 | `model/errors.go` | S3 error XML + WriteS3Error helper |
| 5 | `model/s3xml_test.go` | Unit tests for XML parsing/generation |
| 6 | `gcs/client.go` | BucketOperator interface + real GCS implementation |
| 7 | `converter/versioning.go` | Versioning conversion + validation |
| 8 | `converter/cors.go` | CORS conversion |
| 9 | `converter/logging.go` | Logging conversion + TargetGrants validation |
| 10 | `converter/tagging.go` | Tagging→Labels conversion + key normalization |
| 11 | `converter/converter_test.go` | Unit tests for all converters |
| 12 | `handler/common.go` | Shared XML response writer + GCS error mapping |
| 13 | `handler/versioning.go` | PUT/GET versioning handlers |
| 14 | `handler/cors.go` | PUT/GET/DELETE CORS handlers |
| 15 | `handler/logging.go` | PUT/GET logging handlers (no DELETE — S3 disables logging via PUT with empty body) |
| 16 | `handler/tagging.go` | PUT/GET/DELETE tagging handlers |
| 17 | `server/router.go` | HTTP routing with bucket name extraction |
| 18 | `middleware/signature.go` | S3 v4 signature bypass |
| 19 | `main.go` | Wire everything, graceful shutdown |
| 20 | `handler/handler_test.go` | Integration tests with mock GCS |
| 21 | `server/router_test.go` | Router and bucket extraction tests |
| 22 | `e2e_test.go` | E2E tests with real AWS S3 SDK v2 |

## Testing Strategy

### Unit Tests (`model/s3xml_test.go`, `converter/converter_test.go`)
- XML round-trip: parse sample S3 XML → verify struct fields → marshal back → verify XML output
- Converter: known S3 inputs → verify GCS struct fields, and reverse
- Error cases: MfaDelete enabled → error, TargetGrants present → error, malformed XML → MalformedXML

### Integration Tests (`handler/handler_test.go`)
- Mock `BucketOperator` implementation
- `httptest.NewRecorder()` + `httptest.NewRequest()` for each of 11 endpoints
- Verify HTTP status codes, Content-Type headers, response XML body content
- Test error paths (bucket not found, invalid input)

### Router Tests (`server/router_test.go`)
- Virtual-hosted style bucket name extraction
- Path style bucket name extraction
- Empty/missing bucket name handling

### End-to-End Tests (`e2e_test.go`)
- Start proxy with `httptest.NewServer`
- Configure AWS S3 SDK v2 client: custom endpoint, dummy credentials, path-style addressing
- Call real SDK methods: `PutBucketVersioning`, `GetBucketVersioning`, `PutBucketCors`, etc.
- Verify SDK parses responses correctly (proves full S3 compatibility)
- Uses mock GCS backend (or real GCS with test project)

### Verification Commands
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run E2E tests only
go test -v ./tests/ -run TestE2E

# Manual test with AWS CLI
aws s3api put-bucket-versioning \
  --bucket test-bucket \
  --versioning-configuration Status=Enabled \
  --endpoint-url http://localhost:8080

aws s3api get-bucket-versioning \
  --bucket test-bucket \
  --endpoint-url http://localhost:8080
```

## Build and Run

```bash
# Build
go build -o s3-gcs-proxy .

# Run
export GCS_PROJECT_ID=my-gcp-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
./s3-gcs-proxy --addr :8080
```
