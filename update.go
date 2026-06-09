package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

type initMsg struct {
	client  *s3Con
	buckets []string
	files   []fileItem
	err     error
}

type clearStatusMsg struct{}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func pasteFromClipboard() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	case "windows":
		cmd = exec.Command("powershell", "-command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	viewportHeight := m.visibleRows()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case clearStatusMsg:
		m.statusMsg = ""

	case initMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.s3Client = msg.client
		if m.startBucket != "" {
			m.currentBucket = m.startBucket
			m.files = msg.files
			m.state = fileList
		} else {
			m.buckets = msg.buckets
			m.state = bucketList
		}
		m.cursor = 0
		m.yOffset = 0
		m.currentPrefix = ""

	case tea.KeyMsg:
		// If there's an error, any key quits
		if m.err != nil {
			return m, tea.Quit
		}

		key := msg.String()

		// When help overlay is open, only esc closes it
		if m.showHelp {
			if key == "esc" || key == "?" {
				m.showHelp = false
			}
			return m, nil
		}

		// Search mode input handling
		if m.searching {
			// Bracketed paste arrives as a multi-rune KeyMsg with Paste: true
			if msg.Paste {
				var b strings.Builder
				for _, r := range msg.Runes {
					if unicode.IsPrint(r) {
						b.WriteRune(r)
					}
				}
				m.searchQuery += b.String()
				m = m.recomputeSearchMatches()
				return m, nil
			}
			return m.handleSearchInput(key, viewportHeight)
		}

		// Handle yy (two-press yank)
		if key == "y" {
			if m.pendingY {
				// Second y — build S3 path and copy
				m.pendingY = false
				s3Path := m.buildS3Path()
				if err := copyToClipboard(s3Path); err != nil {
					m.statusMsg = "Failed to copy to clipboard"
				} else {
					m.statusMsg = fmt.Sprintf("Copied: %s", s3Path)
				}
				return m, clearStatusAfter(3 * time.Second)
			}
			// First y — set pending
			m.pendingY = true
			m.statusMsg = "y..."
			return m, nil
		}

		// Any non-y key cancels pending y
		m.pendingY = false
		m.statusMsg = ""

		switch key {
		case "/":
			if m.filterActive {
				m = m.clearFilter()
			}
			m.searching = true
			m.searchQuery = ""
			m.searchMatches = nil
			m.searchCursor = 0
			m = m.recomputeSearchMatches()
			return m, nil
		case "?":
			m.showHelp = true
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.yOffset {
					m.yOffset = m.cursor
				}
			}
		case "down", "j":
			listLen := 0
			if m.state == bucketList {
				listLen = len(m.buckets)
			} else {
				listLen = len(m.files)
			}

			if m.cursor < listLen-1 {
				m.cursor++
				if m.cursor >= m.yOffset+viewportHeight {
					m.yOffset++
				}
			}
		case "enter":
			return m.handleEnter()
		case "esc", "backspace":
			if m.filterActive {
				m = m.clearFilter()
				return m, nil
			}
			if m.state == fileList {
				if m.currentPrefix == "" {
					// Only go back to bucket list if an initial bucket wasn't provided
					if m.startBucket == "" {
						buckets, err := m.s3Client.listBucket()
						if err != nil {
							m.err = err
							return m, nil
						}
						m.buckets = buckets
						m.state = bucketList
						m.cursor = 0
						m.yOffset = 0
					}
				} else {
					parts := strings.Split(strings.TrimSuffix(m.currentPrefix, "/"), "/")
					if len(parts) > 1 {
						m.currentPrefix = strings.Join(parts[:len(parts)-1], "/") + "/"
					} else {
						m.currentPrefix = ""
					}
					files, err := m.s3Client.listPrefix(m.currentBucket, m.currentPrefix)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.files = files
					m.cursor = 0
					m.yOffset = 0
				}
			}
		case "G":
			listLen := 0
			if m.state == bucketList {
				listLen = len(m.buckets)
			} else {
				listLen = len(m.files)
			}
			if listLen > 0 {
				m.cursor = listLen - 1
				if m.cursor >= m.yOffset+viewportHeight {
					m.yOffset = m.cursor - viewportHeight + 1
				}
			}
		case "g":
			m.cursor = 0
			m.yOffset = 0
		case "pgup":
			m.cursor -= viewportHeight
			if m.cursor < 0 {
				m.cursor = 0
			}
			if m.cursor < m.yOffset {
				m.yOffset = m.cursor
			}
		case "pgdown":
			listLen := 0
			if m.state == bucketList {
				listLen = len(m.buckets)
			} else {
				listLen = len(m.files)
			}
			m.cursor += viewportHeight
			if m.cursor >= listLen {
				m.cursor = listLen - 1
			}
			if m.cursor >= m.yOffset+viewportHeight {
				m.yOffset = m.cursor - viewportHeight + 1
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}
	}
	return m, nil
}

// handleSearchInput processes keystrokes while in search mode.
func (m model) handleSearchInput(key string, viewportHeight int) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		// Cancel search, restore original view
		m.searching = false
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchCursor = 0
		return m, nil
	case "enter":
		// Apply the current matches as a persistent filter and exit search mode (no navigation)
		if len(m.searchMatches) > 0 {
			selectedPos := m.searchCursor
			m = m.applyFilter()
			m.cursor = selectedPos
			if m.cursor >= m.yOffset+viewportHeight {
				m.yOffset = m.cursor - viewportHeight + 1
			}
			if m.yOffset < 0 {
				m.yOffset = 0
			}
		}
		m.searching = false
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchCursor = 0
		return m, nil
	case "backspace":
		if len(m.searchQuery) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.searchQuery)
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-size]
			m = m.recomputeSearchMatches()
		}
		return m, nil
	case "ctrl+v":
		text, err := pasteFromClipboard()
		if err == nil && text != "" {
			var b strings.Builder
			for _, r := range text {
				if unicode.IsPrint(r) {
					b.WriteRune(r)
				}
			}
			m.searchQuery += b.String()
			m = m.recomputeSearchMatches()
		}
		return m, nil
	case "up", "ctrl+p":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.searchCursor < len(m.searchMatches)-1 {
			m.searchCursor++
		}
		return m, nil
	default:
		// Append printable characters to query
		if utf8.RuneCountInString(key) == 1 {
			r, _ := utf8.DecodeRuneInString(key)
			if unicode.IsPrint(r) {
				m.searchQuery += key
				m = m.recomputeSearchMatches()
			}
		}
		return m, nil
	}
}

// handleEnter executes the "enter" action (navigate into bucket/folder).
func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.state == bucketList {
		if m.cursor >= len(m.buckets) {
			return m, nil
		}
		m.currentBucket = m.buckets[m.cursor]
		files, err := m.s3Client.listPrefix(m.currentBucket, "")
		if err != nil {
			m.err = err
			return m, nil
		}
		m.files = files
		m.state = fileList
		m.cursor = 0
		m.yOffset = 0
		m.currentPrefix = ""
		// Clear any bucket-level filter since we're now in a different view
		m.filterActive = false
		m.unfilteredBuckets = nil
	} else if m.state == fileList {
		if m.cursor >= len(m.files) {
			return m, nil
		}
		selected := m.files[m.cursor]
		if selected.isDir {
			newPrefix := m.currentPrefix + selected.name
			files, err := m.s3Client.listPrefix(m.currentBucket, newPrefix)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.files = files
			m.currentPrefix = newPrefix
			m.cursor = 0
			m.yOffset = 0
			// Clear filter since we're now in a different directory
			m.filterActive = false
			m.unfilteredFiles = nil
		}
	}
	return m, nil
}

// applyFilter replaces the current list with only the search-matched items.
func (m model) applyFilter() model {
	if m.state == bucketList {
		unfiltered := make([]string, len(m.buckets))
		copy(unfiltered, m.buckets)
		m.unfilteredBuckets = unfiltered
		filtered := make([]string, 0, len(m.searchMatches))
		for _, idx := range m.searchMatches {
			filtered = append(filtered, m.buckets[idx])
		}
		m.buckets = filtered
	} else {
		unfiltered := make([]fileItem, len(m.files))
		copy(unfiltered, m.files)
		m.unfilteredFiles = unfiltered
		filtered := make([]fileItem, 0, len(m.searchMatches))
		for _, idx := range m.searchMatches {
			filtered = append(filtered, m.files[idx])
		}
		m.files = filtered
	}
	m.filterActive = true
	m.yOffset = 0
	return m
}

// clearFilter restores the full unfiltered list.
func (m model) clearFilter() model {
	if m.state == bucketList && m.unfilteredBuckets != nil {
		m.buckets = m.unfilteredBuckets
		m.unfilteredBuckets = nil
	} else if m.state == fileList && m.unfilteredFiles != nil {
		m.files = m.unfilteredFiles
		m.unfilteredFiles = nil
	}
	m.filterActive = false
	m.cursor = 0
	m.yOffset = 0
	return m
}

// recomputeSearchMatches filters the current list by the search query.
func (m model) recomputeSearchMatches() model {
	m.searchMatches = nil
	m.searchCursor = 0

	query := strings.ToLower(m.searchQuery)
	if m.state == bucketList {
		for i, item := range m.buckets {
			if query == "" || strings.Contains(strings.ToLower(item), query) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	} else {
		for i, item := range m.files {
			if query == "" || strings.Contains(strings.ToLower(item.name), query) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	}
	return m
}

// buildS3Path constructs the full s3:// URI for the item under the cursor.
func (m model) buildS3Path() string {
	if m.state == bucketList {
		if m.cursor < len(m.buckets) {
			return fmt.Sprintf("s3://%s", m.buckets[m.cursor])
		}
		return "s3://"
	}
	// fileList state
	if m.cursor < len(m.files) {
		return fmt.Sprintf("s3://%s/%s%s", m.currentBucket, m.currentPrefix, m.files[m.cursor].name)
	}
	return fmt.Sprintf("s3://%s/%s", m.currentBucket, m.currentPrefix)
}
