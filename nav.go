package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchFiles returns a Cmd that fetches a prefix asynchronously.
func fetchFiles(client *s3Con, bucket, prefix string) tea.Cmd {
	return func() tea.Msg {
		files, err := client.listPrefix(bucket, prefix)
		return fetchFilesMsg{bucket: bucket, prefix: prefix, files: files, err: err}
	}
}

// fetchBuckets returns a Cmd that fetches the bucket list asynchronously.
func fetchBuckets(client *s3Con) tea.Cmd {
	return func() tea.Msg {
		buckets, err := client.listBucket()
		return fetchBucketsMsg{buckets: buckets, err: err}
	}
}

// handleEnter navigates into the selected bucket or folder.
// Returns immediately (no loading) on a cache hit; fires an async fetch otherwise.
func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.state == bucketList {
		if m.cursor >= len(m.buckets) {
			return m, nil
		}
		bucket := m.buckets[m.cursor]
		m.filter = filterState{}
		if items, ok := m.s3Client.getCachedFiles(bucket, ""); ok {
			m.files = items
			m.currentBucket = bucket
			m.currentPrefix = ""
			m.state = fileList
			m.cursor = 0
			m.yOffset = 0
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(fetchFiles(m.s3Client, bucket, ""), tickSpinner())
	}

	if m.state == fileList {
		if m.cursor >= len(m.files) {
			return m, nil
		}
		selected := m.files[m.cursor]
		if !selected.isDir {
			return m, nil
		}
		newPrefix := m.currentPrefix + selected.name
		m.filter = filterState{}
		if items, ok := m.s3Client.getCachedFiles(m.currentBucket, newPrefix); ok {
			m.files = items
			m.currentPrefix = newPrefix
			m.cursor = 0
			m.yOffset = 0
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(fetchFiles(m.s3Client, m.currentBucket, newPrefix), tickSpinner())
	}

	return m, nil
}

// handleBack navigates up one level (prefix or back to bucket list).
// Returns immediately on a cache hit; fires an async fetch otherwise.
func (m model) handleBack() (tea.Model, tea.Cmd) {
	if m.state != fileList {
		return m, nil
	}
	m.filter = filterState{}

	if m.currentPrefix == "" {
		// Can't go further back when a start bucket was specified
		if m.startBucket != "" {
			return m, nil
		}
		if buckets, ok := m.s3Client.getCachedBuckets(); ok {
			m.buckets = buckets
			m.state = bucketList
			m.cursor = 0
			m.yOffset = 0
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(fetchBuckets(m.s3Client), tickSpinner())
	}

	parts := strings.Split(strings.TrimSuffix(m.currentPrefix, "/"), "/")
	var newPrefix string
	if len(parts) > 1 {
		newPrefix = strings.Join(parts[:len(parts)-1], "/") + "/"
	}
	if items, ok := m.s3Client.getCachedFiles(m.currentBucket, newPrefix); ok {
		m.files = items
		m.currentPrefix = newPrefix
		m.cursor = 0
		m.yOffset = 0
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(fetchFiles(m.s3Client, m.currentBucket, newPrefix), tickSpinner())
}

// handleForceRefresh evicts the current view from cache and re-fetches.
func (m model) handleForceRefresh() (tea.Model, tea.Cmd) {
	m.filter = filterState{}
	if m.state == bucketList {
		m.s3Client.evictBuckets()
		m.loading = true
		return m, tea.Batch(fetchBuckets(m.s3Client), tickSpinner())
	}
	m.s3Client.evictFiles(m.currentBucket, m.currentPrefix)
	m.loading = true
	return m, tea.Batch(fetchFiles(m.s3Client, m.currentBucket, m.currentPrefix), tickSpinner())
}

// buildS3Path constructs the full s3:// URI for the item under the cursor.
func (m model) buildS3Path() string {
	if m.state == bucketList {
		if m.cursor < len(m.buckets) {
			return fmt.Sprintf("s3://%s", m.buckets[m.cursor])
		}
		return "s3://"
	}
	if m.cursor < len(m.files) {
		return fmt.Sprintf("s3://%s/%s%s", m.currentBucket, m.currentPrefix, m.files[m.cursor].name)
	}
	return fmt.Sprintf("s3://%s/%s", m.currentBucket, m.currentPrefix)
}
