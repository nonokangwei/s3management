# Requirement description
This project design for realizing a S3 management api proxy which cover S3 bucketing versioning, S3 bucket logging, S3 bucket CORS and S3 bucket tagging APIs, this proxy will accept native S3 SDK API call, then convert the request to google cloud storage API call. It makes the Google Cloud GCS storage backend compatible to native S3 SDK API on S3 bucketing versioning, S3 bucket logging, S3 bucket CORS and S3 bucket tagging APIs.

# Why this project
Because Google Cloud native XML APIs are not fully compatible AWS S3 XML API on S3 bucketing versioning, S3 bucket logging, S3 bucket CORS and S3 bucket tagging APIs. Use this proxy will make the Google Cloud GCS storage backend compatible to native S3 SDK API on these APIs. Customer using this proxy can keep the S3 SDK to set the GCS backend on these APIs.

# How
- Using Golang language
- Make sure compatible on S3 bucketing versioning, S3 bucket logging, S3 bucket CORS and S3 bucket tagging APIs
- Use native GCS JSON SDK to communite with GCS backend
- S3 client SDK will call the proxy with v4 Signature, Ignore the Signature check
- S3's API's payload schema is different from GCS's API's payload schema, just extract the GCS suppored field. If S3 request with non-compatible field, proxy response with fail code.
- Add related comments on the code, help to understand