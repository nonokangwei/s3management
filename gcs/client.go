package gcs

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// BucketOperator defines the interface for GCS bucket operations.
// This interface enables mock-based testing of handlers without a real GCS backend.
type BucketOperator interface {
	GetBucketAttrs(ctx context.Context, bucket string) (*storage.BucketAttrs, error)
	UpdateBucket(ctx context.Context, bucket string, attrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error)
}

// Client wraps the GCS storage client and implements BucketOperator.
type Client struct {
	client    *storage.Client
	projectID string
}

// NewClient creates a new GCS client wrapper.
// Uses Application Default Credentials if no additional options are provided.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

// GetBucketAttrs retrieves the attributes of a GCS bucket.
func (c *Client) GetBucketAttrs(ctx context.Context, bucket string) (*storage.BucketAttrs, error) {
	return c.client.Bucket(bucket).Attrs(ctx)
}

// UpdateBucket updates the attributes of a GCS bucket.
func (c *Client) UpdateBucket(ctx context.Context, bucket string, attrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	return c.client.Bucket(bucket).Update(ctx, attrs)
}

// Close closes the underlying GCS client.
func (c *Client) Close() error {
	return c.client.Close()
}
