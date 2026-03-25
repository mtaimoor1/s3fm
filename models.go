package main

import tea "github.com/charmbracelet/bubbletea"

type sessionState int

const (
	bucketList sessionState = iota
	fileList
)

type model struct {
	profile       string
	region        string
	startBucket   string
	s3Client      *s3Con
	buckets       []string
	files         []string
	cursor        int
	state         sessionState
	currentBucket string
	currentPrefix string
	err           error
	width         int
	height        int
	yOffset       int
	pendingY      bool
	statusMsg     string
	showHelp      bool
	searching     bool
	searchQuery   string
	searchMatches []int
	searchCursor  int
}

func (m model) viewportHeight() int {
	// title(1) + breadcrumb(1) + box borders(2) + footer bar(1) = 5
	const chromeHeight = 5
	h := m.height - chromeHeight
	if h < 1 {
		return 1
	}
	return h
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		s3Con, err := newS3Con(m.profile, m.region)
		if err != nil {
			return initMsg{err: err}
		}

		if m.startBucket != "" {
			files, err := s3Con.listPrefix(m.startBucket, "")
			if err != nil {
				return initMsg{client: s3Con, err: err}
			}
			return initMsg{client: s3Con, files: files}
		}

		buckets, err := s3Con.listBucket()
		if err != nil {
			return initMsg{client: s3Con, err: err}
		}
		return initMsg{client: s3Con, buckets: buckets}
	}
}
