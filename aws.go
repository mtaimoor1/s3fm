package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fileItem struct {
	name         string
	size         int64
	lastModified time.Time
	isDir        bool
}

type s3Con struct {
	ctx    context.Context
	cancel context.CancelFunc
	clnt   *s3.Client
}

func newS3Con(name, region string) (*s3Con, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(name),
		config.WithRegion(region))

	if err != nil {
		cancel()
		return nil, fmt.Errorf("unable to get config for s3 client: %w", err)
	}

	return &s3Con{
		ctx:    ctx,
		cancel: cancel,
		clnt:   s3.NewFromConfig(cfg),
	}, nil
}

func (s *s3Con) close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *s3Con) listBucket() ([]string, error) {
	out, err := s.clnt.ListBuckets(s.ctx, nil)
	var result []string
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	for _, b := range out.Buckets {
		result = append(result, aws.ToString(b.Name))
	}
	return result, nil
}

func (s *s3Con) listPrefix(bucket string, prefix string) ([]fileItem, error) {
	var token *string
	var result []fileItem

	for {
		param := &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
			Delimiter:         aws.String("/"),
		}
		out, err := s.clnt.ListObjectsV2(s.ctx, param)
		if err != nil {
			return nil, fmt.Errorf("unable to list prefix %s: %w", prefix, err)
		}

		for _, v := range out.CommonPrefixes {
			result = append(result, fileItem{
				name:  strings.TrimPrefix(aws.ToString(v.Prefix), prefix),
				isDir: true,
			})
		}
		for _, v := range out.Contents {
			name := strings.TrimPrefix(aws.ToString(v.Key), prefix)
			if name == "" {
				continue
			}
			var modTime time.Time
			if v.LastModified != nil {
				modTime = *v.LastModified
			}
			result = append(result, fileItem{
				name:         name,
				size:         aws.ToInt64(v.Size),
				lastModified: modTime,
			})
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return result, nil
}
