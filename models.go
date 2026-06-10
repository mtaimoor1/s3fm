package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type sessionState int

const (
	bucketList sessionState = iota
	fileList
)

// searchState holds all state for the active search overlay.
type searchState struct {
	active  bool
	query   string
	matches []int
	cursor  int
}

// filterState holds the persistent filter applied after confirming a search.
type filterState struct {
	active            bool
	unfilteredBuckets []string
	unfilteredFiles   []fileItem
}

type model struct {
	// AWS config
	profile     string
	region      string
	startBucket string
	s3Client    *s3Con

	// list data
	buckets []string
	files   []fileItem

	// navigation
	state         sessionState
	currentBucket string
	currentPrefix string

	// cursor / viewport
	cursor  int
	yOffset int
	width   int
	height  int

	// search & filter
	search searchState
	filter filterState

	// UI state
	loading      bool
	spinnerFrame int
	pendingY     bool
	statusMsg    string
	showHelp     bool
	err          error
}

// --- Message types ---

type initMsg struct {
	client  *s3Con
	buckets []string
	files   []fileItem
	err     error
}

type clearStatusMsg struct{}

type fetchFilesMsg struct {
	bucket string
	prefix string
	files  []fileItem
	err    error
}

type fetchBucketsMsg struct {
	buckets []string
	err     error
}

type spinnerTickMsg struct{}

// --- Spinner ---

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func tickSpinner() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// --- Viewport helpers ---

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
		vp -= 2
		if vp < 1 {
			return 1
		}
	}
	return vp
}

func (m model) currentListLen() int {
	if m.state == bucketList {
		return len(m.buckets)
	}
	return len(m.files)
}

// clampScroll ensures yOffset keeps the cursor visible.
func (m model) clampScroll() model {
	rows := m.visibleRows()
	if m.cursor < m.yOffset {
		m.yOffset = m.cursor
	} else if m.cursor >= m.yOffset+rows {
		m.yOffset = m.cursor - rows + 1
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}
	return m
}

func (m model) moveUp() model {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.yOffset {
			m.yOffset = m.cursor
		}
	}
	return m
}

func (m model) moveDown() model {
	listLen := m.currentListLen()
	if m.cursor < listLen-1 {
		m.cursor++
		rows := m.visibleRows()
		if m.cursor >= m.yOffset+rows {
			m.yOffset++
		}
	}
	return m
}

func (m model) jumpTop() model {
	m.cursor = 0
	m.yOffset = 0
	return m
}

func (m model) jumpBottom() model {
	listLen := m.currentListLen()
	if listLen > 0 {
		m.cursor = listLen - 1
		m = m.clampScroll()
	}
	return m
}

func (m model) pageUp() model {
	rows := m.visibleRows()
	m.cursor -= rows
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor < m.yOffset {
		m.yOffset = m.cursor
	}
	return m
}

func (m model) pageDown() model {
	rows := m.visibleRows()
	listLen := m.currentListLen()
	m.cursor += rows
	if m.cursor >= listLen {
		m.cursor = listLen - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	return m.clampScroll()
}

// --- Init ---

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickSpinner(),
		func() tea.Msg {
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
		},
	)
}
