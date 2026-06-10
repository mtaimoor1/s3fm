package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	cacheTTL     = 30 * time.Second
	cacheMaxSize = 500
)

type fileItem struct {
	name         string
	size         int64
	lastModified time.Time
	isDir        bool
}

type fileCacheEntry struct {
	items     []fileItem
	fetchedAt time.Time
}

type s3Con struct {
	ctx           context.Context
	cancel        context.CancelFunc
	clnt          *s3.Client
	mu            sync.Mutex
	fileCache     map[string]fileCacheEntry
	bucketCache   []string
	bucketCacheAt time.Time
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
		ctx:       ctx,
		cancel:    cancel,
		clnt:      s3.NewFromConfig(cfg),
		fileCache: make(map[string]fileCacheEntry),
	}, nil
}

func (s *s3Con) close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *s3Con) getCachedFiles(bucket, prefix string) ([]fileItem, bool) {
	key := bucket + "\x00" + prefix
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.fileCache[key]
	if !ok || time.Since(entry.fetchedAt) > cacheTTL {
		return nil, false
	}
	items := make([]fileItem, len(entry.items))
	copy(items, entry.items)
	return items, true
}

func (s *s3Con) setCachedFiles(bucket, prefix string, items []fileItem) {
	key := bucket + "\x00" + prefix
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.fileCache) >= cacheMaxSize {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range s.fileCache {
			if oldestKey == "" || v.fetchedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.fetchedAt
			}
		}
		delete(s.fileCache, oldestKey)
	}
	s.fileCache[key] = fileCacheEntry{items: items, fetchedAt: time.Now()}
}

func (s *s3Con) evictFiles(bucket, prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fileCache, bucket+"\x00"+prefix)
}

func (s *s3Con) getCachedBuckets() ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bucketCache == nil || time.Since(s.bucketCacheAt) > cacheTTL {
		return nil, false
	}
	cached := make([]string, len(s.bucketCache))
	copy(cached, s.bucketCache)
	return cached, true
}

func (s *s3Con) evictBuckets() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bucketCache = nil
	s.bucketCacheAt = time.Time{}
}

func (s *s3Con) listBucket() ([]string, error) {
	if cached, ok := s.getCachedBuckets(); ok {
		return cached, nil
	}
	out, err := s.clnt.ListBuckets(s.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	var result []string
	for _, b := range out.Buckets {
		result = append(result, aws.ToString(b.Name))
	}
	s.mu.Lock()
	s.bucketCache = result
	s.bucketCacheAt = time.Now()
	s.mu.Unlock()
	return result, nil
}

func (s *s3Con) listPrefix(bucket, prefix string) ([]fileItem, error) {
	if items, ok := s.getCachedFiles(bucket, prefix); ok {
		return items, nil
	}
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
	s.setCachedFiles(bucket, prefix, result)
	return result, nil
}
