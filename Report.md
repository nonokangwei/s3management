# S3-to-GCS Proxy Test Report

**Date:** 2026-03-20
**GCS Test Bucket:** `s3managementtest`
**Go Version:** 1.25
**Overall Result:** ALL PASS (45/45 tests)

---

## Summary

| Test Level | Package | Tests | Passed | Failed | Duration |
|------------|---------|-------|--------|--------|----------|
| Unit | `model` | 9 | 9 | 0 | 0.475s |
| Unit | `converter` | 18 | 18 | 0 | 0.533s |
| Unit | `server` | 5 | 5 | 0 | 0.671s |
| Integration | `handler` | 16 | 16 | 0 | 1.068s |
| E2E | `main` (e2e_test.go) | 5 | 5 | 0 | 12.051s |
| **Total** | | **45** | **45** | **0** | |

---

## 1. Unit Tests - XML Parsing/Generation (`model`)

Tests that S3 XML request/response structs correctly parse and marshal XML, including namespace handling.

| Test | Description | Result |
|------|-------------|--------|
| TestVersioningConfigurationXML/versioning_enabled | Parse `<Status>Enabled</Status>` | PASS |
| TestVersioningConfigurationXML/versioning_suspended | Parse `<Status>Suspended</Status>` | PASS |
| TestVersioningConfigurationXML/versioning_with_MfaDelete | Parse MfaDelete field | PASS |
| TestVersioningConfigurationXML/empty_versioning | Parse empty VersioningConfiguration | PASS |
| TestCORSConfigurationXML | Parse multi-origin, multi-method CORS rule with all fields | PASS |
| TestBucketLoggingStatusXML/logging_enabled | Parse LoggingEnabled with TargetBucket/TargetPrefix | PASS |
| TestBucketLoggingStatusXML/logging_disabled | Parse empty BucketLoggingStatus | PASS |
| TestTaggingXML | Parse TagSet with 2 tags, round-trip marshal | PASS |
| TestVersioningConfigurationWithNamespace | Parse XML with S3 namespace (`xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`) | PASS |
| TestBucketLoggingWithTargetGrants | Parse logging XML with TargetGrants (unsupported field detection) | PASS |

## 2. Unit Tests - S3/GCS Conversion (`converter`)

Tests the bidirectional conversion between S3 XML models and GCS SDK types.

| Test | Description | Result |
|------|-------------|--------|
| TestVersioningToGCS_Enabled | S3 `Enabled` -> GCS `VersioningEnabled: true` | PASS |
| TestVersioningToGCS_Suspended | S3 `Suspended` -> GCS `VersioningEnabled: false` | PASS |
| TestVersioningToGCS_MfaDeleteError | MfaDelete `Enabled` -> error | PASS |
| TestVersioningToGCS_InvalidStatus | Invalid status string -> error | PASS |
| TestVersioningFromGCS_Enabled | GCS `VersioningEnabled: true` -> S3 `Enabled` | PASS |
| TestVersioningFromGCS_Disabled | GCS `VersioningEnabled: false` -> S3 `Suspended` | PASS |
| TestCORSToGCS | S3 CORS rule -> GCS CORS (Origins, Methods, ResponseHeaders, MaxAge) | PASS |
| TestCORSFromGCS | GCS CORS -> S3 CORS rule (reverse mapping) | PASS |
| TestCORSToGCS_EmptyRules | Empty S3 CORS -> empty GCS CORS slice | PASS |
| TestLoggingToGCS | S3 LoggingEnabled -> GCS BucketLogging (LogBucket, LogObjectPrefix) | PASS |
| TestLoggingToGCS_DisableLogging | Nil LoggingEnabled -> empty GCS BucketLogging (disable) | PASS |
| TestLoggingToGCS_TargetGrantsError | TargetGrants present -> error | PASS |
| TestLoggingFromGCS | GCS BucketLogging -> S3 LoggingEnabled | PASS |
| TestLoggingFromGCS_NoLogging | Nil GCS Logging -> nil LoggingEnabled | PASS |
| TestTaggingToGCS | S3 Tags -> GCS labels (with key lowercasing) | PASS |
| TestTaggingDeleteToGCS | Build delete-all-labels update from current labels | PASS |
| TestTaggingFromGCS | GCS labels -> S3 Tags | PASS |
| TestTaggingFromGCS_NoLabels | Nil GCS labels -> empty TagSet | PASS |

## 3. Unit Tests - Router (`server`)

Tests bucket name extraction from HTTP requests in both addressing styles.

| Test | Description | Result |
|------|-------------|--------|
| TestExtractBucketName_PathStyle/path_style_simple | `/mybucket` on `localhost:8080` | PASS |
| TestExtractBucketName_PathStyle/path_style_with_trailing_slash | `/mybucket/` on `localhost:8080` | PASS |
| TestExtractBucketName_PathStyle/path_style_with_query | `/mybucket` on `s3.amazonaws.com` | PASS |
| TestExtractBucketName_VirtualHosted | `/` on `mybucket.s3.amazonaws.com` | PASS |
| TestExtractBucketName_Empty | `/` on `localhost:8080` -> empty string | PASS |

## 4. Integration Tests - Handlers (`handler`)

Tests all HTTP handlers with a mock GCS client, verifying status codes, Content-Type headers, and XML response bodies.

| Test | Description | Result |
|------|-------------|--------|
| TestGetVersioning | GET /?versioning -> 200, XML with `Enabled` | PASS |
| TestPutVersioning_Enabled | PUT /?versioning with Enabled -> 200 | PASS |
| TestPutVersioning_MfaDeleteError | PUT with MfaDelete -> 400 InvalidArgument | PASS |
| TestPutVersioning_MalformedXML | PUT with invalid XML -> 400 MalformedXML | PASS |
| TestGetVersioning_BucketNotFound | GET on nonexistent bucket -> 404 NoSuchBucket | PASS |
| TestGetCORS | GET /?cors -> 200, XML with CORS rules | PASS |
| TestPutCORS | PUT /?cors with CORSConfiguration -> 200 | PASS |
| TestDeleteCORS | DELETE /?cors -> 204 | PASS |
| TestGetLogging | GET /?logging -> 200, XML with LoggingEnabled | PASS |
| TestPutLogging | PUT /?logging with LoggingEnabled -> 200 | PASS |
| TestPutLogging_TargetGrantsError | PUT with TargetGrants -> 400 InvalidArgument | PASS |
| TestPutLogging_DisableLogging | PUT with empty BucketLoggingStatus -> 200 (disable) | PASS |
| TestGetTagging | GET /?tagging -> 200, XML with 2 tags | PASS |
| TestPutTagging | PUT /?tagging with TagSet -> 200 | PASS |
| TestDeleteTagging | DELETE /?tagging -> 204 | PASS |
| TestDeleteTagging_NoLabels | DELETE /?tagging with no existing labels -> 204 | PASS |

## 5. End-to-End Tests (`e2e_test.go`)

Tests the full proxy stack against the real GCS bucket `s3managementtest` using the AWS S3 SDK v2 Go client. The proxy runs on `httptest.NewServer`, and the S3 SDK client uses path-style addressing with dummy credentials.

### TestE2E_Versioning (2.78s) - PASS

Full lifecycle test of bucket versioning.

| Step | S3 SDK Call | Proxy Route | GCS Operation | Assertion |
|------|------------|-------------|---------------|-----------|
| 1 | `PutBucketVersioning(Status=Enabled)` | PUT /?versioning | `bucket.Update(VersioningEnabled: true)` | No error |
| 2 | `GetBucketVersioning` | GET /?versioning | `bucket.Attrs()` | Status == `Enabled` |
| 3 | `PutBucketVersioning(Status=Suspended)` | PUT /?versioning | `bucket.Update(VersioningEnabled: false)` | No error |
| 4 | `GetBucketVersioning` | GET /?versioning | `bucket.Attrs()` | Status == `Suspended` |

### TestE2E_CORS (3.07s) - PASS

Full lifecycle test of bucket CORS configuration.

| Step | S3 SDK Call | Proxy Route | GCS Operation | Assertion |
|------|------------|-------------|---------------|-----------|
| 1 | `PutBucketCors(2 origins, 2 methods, headers, MaxAge=3600)` | PUT /?cors | `bucket.Update(CORS: [...])` | No error |
| 2 | `GetBucketCors` | GET /?cors | `bucket.Attrs()` | 1 rule, 2 origins, 2 methods |
| 3 | `DeleteBucketCors` | DELETE /?cors | `bucket.Update(CORS: [])` | No error |
| 4 | `GetBucketCors` | GET /?cors | `bucket.Attrs()` | 0 rules |

### TestE2E_Tagging (2.98s) - PASS

Full lifecycle test of bucket tagging (mapped to GCS labels).

| Step | S3 SDK Call | Proxy Route | GCS Operation | Assertion |
|------|------------|-------------|---------------|-----------|
| 1 | `PutBucketTagging(environment=test, project=s3management)` | PUT /?tagging | `bucket.Update(SetLabel...)` | No error |
| 2 | `GetBucketTagging` | GET /?tagging | `bucket.Attrs()` | 2 tags with correct values |
| 3 | `DeleteBucketTagging` | DELETE /?tagging | `bucket.Update(DeleteLabel...)` | No error |
| 4 | `GetBucketTagging` | GET /?tagging | `bucket.Attrs()` | 0 tags |

### TestE2E_Logging (2.53s) - PASS

Full lifecycle test of bucket logging configuration.

| Step | S3 SDK Call | Proxy Route | GCS Operation | Assertion |
|------|------------|-------------|---------------|-----------|
| 1 | `PutBucketLogging(TargetBucket=s3managementtest, TargetPrefix=logs/)` | PUT /?logging | `bucket.Update(Logging: {...})` | No error |
| 2 | `GetBucketLogging` | GET /?logging | `bucket.Attrs()` | TargetBucket and TargetPrefix match |
| 3 | `PutBucketLogging(empty BucketLoggingStatus)` | PUT /?logging | `bucket.Update(Logging: {})` | No error (disable) |
| 4 | `GetBucketLogging` | GET /?logging | `bucket.Attrs()` | LoggingEnabled == nil |

### TestE2E_Versioning_MfaDeleteError (0.00s) - PASS

Validates that the unsupported MfaDelete field is correctly rejected.

| Step | S3 SDK Call | Expected | Actual |
|------|------------|----------|--------|
| 1 | `PutBucketVersioning(Status=Enabled, MFADelete=Enabled)` | 400 InvalidArgument | `api error InvalidArgument: MfaDelete is not supported by GCS` |

---

## Notes

- **S3 Logging has no DELETE API**: S3 disables logging via `PUT /?logging` with an empty `BucketLoggingStatus` body (no `LoggingEnabled` element). This is correctly handled by the proxy.
- **GCS label keys are lowercase**: The proxy automatically lowercases S3 tag keys when converting to GCS labels.
- **S3 v4 signature**: Detected in all E2E requests and correctly bypassed as per design requirements.
- **Unsupported fields**: `MfaDelete` (versioning) and `TargetGrants` (logging) are correctly rejected with S3-compatible `InvalidArgument` error responses.
