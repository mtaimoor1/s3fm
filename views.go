package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Reverse(true).
				Bold(true).
				PaddingLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Padding(1)
)

func (m model) View() string {
	if m.err != nil {
		// Wrap error text to width
		w := m.width
		if w == 0 {
			w = 80 // Default fallback width
		}

		errText := fmt.Sprintf("Error: %v", m.err)
		wrappedErr := errorStyle.Width(w - 4).Render(errText)

		return fmt.Sprintf("%s\n\nPress q to quit.", wrappedErr)
	}

	// Calculate heights
	// Header: Text + Empty Line = 2 lines
	const headerHeight = 3
	// Footer: Empty Line + Text = 2 lines
	const footerHeight = 2

	viewportHeight := m.height - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	s := ""
	if m.state == bucketList {
		header := fmt.Sprintf("Region: %s\n Select a bucket:", m.region)
		s += titleStyle.Render(header) + "\n\n"

		start := m.yOffset
		end := start + viewportHeight
		if end > len(m.buckets) {
			end = len(m.buckets)
		}

		for i := start; i < end; i++ {
			b := m.buckets[i]
			if m.cursor == i {
				s += selectedItemStyle.Render(fmt.Sprintf("[x] %s", b)) + "\n"
			} else {
				s += itemStyle.Render(fmt.Sprintf("[ ] %s", b)) + "\n"
			}
		}
	} else {
		header := fmt.Sprintf("Region: %s Bucket: %s, Path: %s", m.region, m.currentBucket, m.currentPrefix)
		// Truncate header if too long to prevent wrapping
		if len(header) > m.width-4 { // -4 for padding safety
			if m.width > 7 {
				header = header[:m.width-7] + "..."
			}
		}
		s += titleStyle.Render(header) + "\n\n"

		start := m.yOffset
		end := start + viewportHeight
		if end > len(m.files) {
			end = len(m.files)
		}

		for i := start; i < end; i++ {
			f := m.files[i]
			if m.cursor == i {
				s += selectedItemStyle.Render(fmt.Sprintf("[x] %s", f)) + "\n"
			} else {
				s += itemStyle.Render(fmt.Sprintf("[ ] %s", f)) + "\n"
			}
		}
	}

	// Add Footer
	// Previous loop adds a newline at the end of the last item.
	// So just adding "Press q..." puts it on the next line.
	// We want 1 empty line before footer.

	s += "\nPress q to quit."
	// Debug info on the same line to save space, or careful with next line
	// Let's put debug info on the same line if it fits, or assume footerHeight covers it.
	// Given we set footerHeight=2, we have:
	// Line N (Last Item) \n
	// Line N+1 (Empty)
	// Line N+2 (Text)
	// So we need one explicit \n before text.
	// Wait, the loop adds \n after the last item.
	// So cursor is at start of Line N+1.
	// "\nText" -> Empty line at N+1, Text at N+2.
	// This matches footerHeight = 2 (space consumption).

	s += fmt.Sprintf(" | Debug: Cur=%d, Off=%d, H=%d", m.cursor, m.yOffset, m.height)

	return s
}
