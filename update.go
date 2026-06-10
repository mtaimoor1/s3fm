package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinnerTickMsg:
		if m.loading {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, tickSpinner()
		}

	case clearStatusMsg:
		m.statusMsg = ""

	case initMsg:
		m.loading = false
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

	case fetchBucketsMsg:
		if !m.loading {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.buckets = msg.buckets
		m.state = bucketList
		m.cursor = 0
		m.yOffset = 0

	case fetchFilesMsg:
		if !m.loading {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.files = msg.files
		m.currentBucket = msg.bucket
		m.currentPrefix = msg.prefix
		m.state = fileList
		m.cursor = 0
		m.yOffset = 0

	case tea.KeyMsg:
		if m.err != nil {
			return m, tea.Quit
		}

		// While loading, only allow quit
		if m.loading {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		key := msg.String()

		// Help overlay: only esc/? closes it
		if m.showHelp {
			if key == "esc" || key == "?" {
				m.showHelp = false
			}
			return m, nil
		}

		// Search mode
		if m.search.active {
			if msg.Paste {
				var b strings.Builder
				for _, r := range msg.Runes {
					if unicode.IsPrint(r) {
						b.WriteRune(r)
					}
				}
				m.search.query += b.String()
				m = m.recomputeSearchMatches()
				return m, nil
			}
			return m.handleSearchInput(key)
		}

		// yy: two-press yank
		if key == "y" {
			if m.pendingY {
				m.pendingY = false
				s3Path := m.buildS3Path()
				if err := copyToClipboard(s3Path); err != nil {
					m.statusMsg = "Failed to copy to clipboard"
				} else {
					m.statusMsg = fmt.Sprintf("Copied: %s", s3Path)
				}
				return m, clearStatusAfter(3 * time.Second)
			}
			m.pendingY = true
			m.statusMsg = "y..."
			return m, nil
		}

		m.pendingY = false
		m.statusMsg = ""

		switch key {
		case "/":
			if m.filter.active {
				m = m.clearFilter()
			}
			m.search = searchState{active: true}
			m = m.recomputeSearchMatches()
			return m, nil
		case "?":
			m.showHelp = true
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m = m.moveUp()
		case "down", "j":
			m = m.moveDown()
		case "enter":
			return m.handleEnter()
		case "esc", "backspace":
			if m.filter.active {
				m = m.clearFilter()
				return m, nil
			}
			return m.handleBack()
		case "G":
			m = m.jumpBottom()
		case "g":
			m = m.jumpTop()
		case "pgup":
			m = m.pageUp()
		case "pgdown":
			m = m.pageDown()
		case "r":
			return m.handleForceRefresh()
		}
	}
	return m, nil
}
