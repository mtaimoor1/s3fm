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
	files         []fileItem
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
	// header box: logo(5) + divider(1) + breadcrumb(1) + borders(2) = 9
	// list box borders(2) + footer(1) = 3
	// total chrome = 12
	const chromeHeight = 12
	h := m.height - chromeHeight
	if h < 1 {
		return 1
	}
	return h
}

// visibleRows returns the number of scrollable data rows in the list box.
// In file list view the grid header+divider take 2 lines from the viewport.
func (m model) visibleRows() int {
	vp := m.viewportHeight()
	if m.state == fileList {
		vp -= 2 // grid header + divider
		if vp < 1 {
			return 1
		}
	}
	return vp
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
