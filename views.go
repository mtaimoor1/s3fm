package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	purple    = lipgloss.Color("#7C3AED")
	violet    = lipgloss.Color("#A78BFA")
	indigo    = lipgloss.Color("#6366F1")
	slate     = lipgloss.Color("#94A3B8")
	darkSlate = lipgloss.Color("#475569")
	white     = lipgloss.Color("#F8FAFC")
	dimWhite  = lipgloss.Color("#CBD5E1")
	green     = lipgloss.Color("#34D399")
	amber     = lipgloss.Color("#FBBF24")
	red       = lipgloss.Color("#F87171")
	darkBg    = lipgloss.Color("#1E1B2E")
	headerBg  = lipgloss.Color("#312E81")
	footerBg  = lipgloss.Color("#1E1B2E")
)

// Styles
var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(amber)

	titleBarStyle = lipgloss.NewStyle().
			Background(headerBg).
			Foreground(white).
			Bold(true).
			Padding(0, 1)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(slate)

	breadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(violet).
				Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(darkSlate)

	itemStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			PaddingLeft(1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(white).
				Bold(true).
				Background(lipgloss.Color("#2D2B55")).
				PaddingLeft(1)

	folderIcon      = lipgloss.NewStyle().Foreground(amber).Render("\U0001F4C1 ")
	fileIcon        = lipgloss.NewStyle().Foreground(slate).Render("   ")
	bucketIcon      = lipgloss.NewStyle().Foreground(green).Render("\U0001F4E6 ")
	selectedPointer = lipgloss.NewStyle().Foreground(purple).Bold(true).Render("\u25B8 ")

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(violet).
			Bold(true)

	statusDescStyle = lipgloss.NewStyle().
			Foreground(slate)

	countStyle = lipgloss.NewStyle().
			Foreground(darkSlate)

	errorBoxStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red).
			Padding(1, 2)

	emptyStyle = lipgloss.NewStyle().
			Foreground(darkSlate).
			Italic(true).
			PaddingLeft(2)
)

func (m model) View() string {
	w := m.width
	if w == 0 {
		w = 80
	}
	h := m.height
	if h == 0 {
		h = 24
	}

	if m.err != nil {
		errText := fmt.Sprintf("  Error: %v", m.err)
		box := errorBoxStyle.Width(w - 6).Render(errText)
		hint := lipgloss.NewStyle().Foreground(slate).PaddingLeft(2).Render("Press any key to quit.")
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("%s\n\n%s", box, hint))
	}

	vpHeight := m.viewportHeight()

	// --- Header ---
	header := m.renderHeader(w)

	// --- List content ---
	var listContent string
	if m.searching {
		listContent = m.renderSearchContent(vpHeight)
	} else if m.state == bucketList {
		listContent = m.renderBucketContent(vpHeight)
	} else {
		listContent = m.renderFileContent(vpHeight)
	}

	// Wrap list in a bordered box
	listBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(darkSlate).
		Width(w - 2)

	listBox := listBoxStyle.Render(listContent)

	// --- Footer ---
	footer := m.renderStatusBar(w)

	// --- Compose full layout ---
	view := lipgloss.JoinVertical(lipgloss.Left, header, listBox, footer)

	// --- Help overlay ---
	if m.showHelp {
		view = m.renderHelpOverlay(w)
	}

	return view
}

// renderHeader renders the title bar and breadcrumb.
func (m model) renderHeader(w int) string {
	logo := logoStyle.Render(" s3fm")
	region := lipgloss.NewStyle().Foreground(slate).Render("  " + m.region)
	titleContent := logo + region

	titleBar := titleBarStyle.Width(w).Render(titleContent)

	// Breadcrumb
	var crumb string
	if m.state == bucketList {
		label := breadcrumbActiveStyle.Render("  Buckets")
		cnt := countStyle.Render(fmt.Sprintf("  %d items", len(m.buckets)))
		crumb = label + cnt
	} else {
		parts := []string{}
		if m.startBucket == "" {
			parts = append(parts, breadcrumbStyle.Render("  Buckets"))
			parts = append(parts, separatorStyle.Render(" \u203A "))
		} else {
			parts = append(parts, breadcrumbStyle.Render("  "))
		}
		parts = append(parts, breadcrumbActiveStyle.Render(m.currentBucket))
		if m.currentPrefix != "" {
			segments := strings.Split(strings.TrimSuffix(m.currentPrefix, "/"), "/")
			for _, seg := range segments {
				parts = append(parts, separatorStyle.Render(" \u203A "))
				parts = append(parts, breadcrumbActiveStyle.Render(seg))
			}
		}
		cnt := countStyle.Render(fmt.Sprintf("  %d items", len(m.files)))
		crumb = strings.Join(parts, "") + cnt
	}

	return titleBar + "\n" + crumb
}

// padToHeight pads content with empty lines to exactly `height` lines.
func padToHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// renderBucketContent returns the bucket list content padded to viewport height.
func (m model) renderBucketContent(vpHeight int) string {
	if len(m.buckets) == 0 {
		return padToHeight(emptyStyle.Render("No buckets found."), vpHeight)
	}

	var lines []string
	start := m.yOffset
	end := start + vpHeight
	if end > len(m.buckets) {
		end = len(m.buckets)
	}

	for i := start; i < end; i++ {
		name := m.buckets[i]
		if m.cursor == i {
			lines = append(lines, selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, bucketIcon, name)))
		} else {
			lines = append(lines, itemStyle.Render(fmt.Sprintf("  %s%s", bucketIcon, name)))
		}
	}

	return padToHeight(strings.Join(lines, "\n"), vpHeight)
}

// renderFileContent returns the file list content padded to viewport height.
func (m model) renderFileContent(vpHeight int) string {
	if len(m.files) == 0 {
		return padToHeight(emptyStyle.Render("This directory is empty."), vpHeight)
	}

	var lines []string
	start := m.yOffset
	end := start + vpHeight
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		name := m.files[i]
		isDir := strings.HasSuffix(name, "/")
		icon := fileIcon
		if isDir {
			icon = folderIcon
		}

		if m.cursor == i {
			lines = append(lines, selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, icon, name)))
		} else {
			lines = append(lines, itemStyle.Render(fmt.Sprintf("  %s%s", icon, name)))
		}
	}

	return padToHeight(strings.Join(lines, "\n"), vpHeight)
}

// renderSearchContent returns the search results padded to viewport height.
func (m model) renderSearchContent(vpHeight int) string {
	if len(m.searchMatches) == 0 {
		return padToHeight(emptyStyle.Render("No matches found."), vpHeight)
	}

	var list []string
	if m.state == bucketList {
		list = m.buckets
	} else {
		list = m.files
	}

	var lines []string
	start := 0
	if m.searchCursor >= vpHeight {
		start = m.searchCursor - vpHeight + 1
	}
	end := start + vpHeight
	if end > len(m.searchMatches) {
		end = len(m.searchMatches)
	}

	for i := start; i < end; i++ {
		realIdx := m.searchMatches[i]
		name := list[realIdx]
		isDir := strings.HasSuffix(name, "/")
		icon := fileIcon
		if m.state == bucketList {
			icon = bucketIcon
		} else if isDir {
			icon = folderIcon
		}

		if i == m.searchCursor {
			lines = append(lines, selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, icon, name)))
		} else {
			lines = append(lines, itemStyle.Render(fmt.Sprintf("  %s%s", icon, name)))
		}
	}

	return padToHeight(strings.Join(lines, "\n"), vpHeight)
}

// renderStatusBar renders the full-width footer bar pinned to the bottom.
func (m model) renderStatusBar(w int) string {
	barStyle := lipgloss.NewStyle().
		Background(footerBg).
		Foreground(slate).
		Width(w).
		Padding(0, 1)

	var left string

	if m.searching {
		prompt := lipgloss.NewStyle().Foreground(violet).Bold(true).Background(footerBg).Render("/")
		query := lipgloss.NewStyle().Foreground(white).Background(footerBg).Render(m.searchQuery)
		cursor := lipgloss.NewStyle().Foreground(violet).Bold(true).Background(footerBg).Render("\u2588")
		matchInfo := ""
		if m.searchQuery != "" {
			matchInfo = lipgloss.NewStyle().Foreground(darkSlate).Background(footerBg).
				Render(fmt.Sprintf("  %d match(es)", len(m.searchMatches)))
		}
		left = prompt + query + cursor + matchInfo
	} else if m.statusMsg != "" {
		left = lipgloss.NewStyle().Foreground(green).Bold(true).Background(footerBg).Render(m.statusMsg)
	} else {
		keys := []struct{ key, desc string }{
			{"j/k", "nav"},
			{"enter", "open"},
			{"esc", "back"},
			{"yy", "copy"},
			{"/", "search"},
			{"G/g", "top/btm"},
			{"?", "help"},
			{"q", "quit"},
		}
		var hints []string
		sep := lipgloss.NewStyle().Foreground(darkSlate).Background(footerBg).Render(" \u2502 ")
		for _, k := range keys {
			hint := lipgloss.NewStyle().Foreground(violet).Bold(true).Background(footerBg).Render(k.key) +
				lipgloss.NewStyle().Foreground(slate).Background(footerBg).Render(" "+k.desc)
			hints = append(hints, hint)
		}
		left = strings.Join(hints, sep)
	}

	// Right side: position indicator
	listLen := len(m.buckets)
	if m.state == fileList {
		listLen = len(m.files)
	}

	var right string
	if listLen > 0 {
		pos := fmt.Sprintf("%d/%d", m.cursor+1, listLen)
		vpHeight := m.viewportHeight()
		if m.yOffset > 0 || m.yOffset+vpHeight < listLen {
			pct := 0
			if listLen > 1 {
				pct = m.cursor * 100 / (listLen - 1)
			}
			pos += fmt.Sprintf(" %d%%", pct)
		}
		right = lipgloss.NewStyle().Foreground(slate).Background(footerBg).Render(pos)
	}

	// Fill middle space
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	innerWidth := w - 2 // account for bar padding
	gap := innerWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	filler := strings.Repeat(" ", gap)

	content := left + filler + right
	return barStyle.Render(content)
}

func (m model) renderHelpOverlay(w int) string {
	const keyColWidth = 20
	const boxWidth = 54
	descColWidth := boxWidth - keyColWidth - 6 // 6 = border(2) + padding(4)

	helpKeyStyle := lipgloss.NewStyle().
		Foreground(violet).
		Bold(true).
		Width(keyColWidth).
		Align(lipgloss.Right).
		PaddingRight(2)

	helpDescStyle := lipgloss.NewStyle().
		Foreground(dimWhite).
		Width(descColWidth)

	helpTitleStyle := lipgloss.NewStyle().
		Foreground(white).
		Bold(true).
		MarginBottom(1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(darkSlate)

	keys := []struct{ key, desc string }{
		{"j / k / arrow", "Move cursor down / up"},
		{"enter", "Open bucket or folder"},
		{"esc / backspace", "Go back"},
		{"G", "Jump to top of list"},
		{"g", "Jump to bottom of list"},
		{"yy", "Copy S3 path to clipboard"},
		{"/", "Search and filter list"},
		{"pgup / pgdown", "Page up / down"},
		{"?", "Toggle this help"},
		{"q / ctrl+c", "Quit"},
	}

	var rows []string
	for _, k := range keys {
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			helpKeyStyle.Render(k.key),
			helpDescStyle.Render(k.desc),
		)
		rows = append(rows, row)
	}

	title := helpTitleStyle.Render("Keyboard Shortcuts")
	content := title + "\n" + strings.Join(rows, "\n")

	footer := lipgloss.NewStyle().
		Foreground(darkSlate).
		Italic(true).
		MarginTop(1).
		Render("Press esc to close")

	content += "\n" + footer

	effectiveBoxWidth := boxWidth
	if w > 0 && w < effectiveBoxWidth+4 {
		effectiveBoxWidth = w - 4
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 2).
		Width(effectiveBoxWidth)

	rendered := box.Render(content)

	return lipgloss.Place(
		w, m.height,
		lipgloss.Center, lipgloss.Center,
		rendered,
	)
}
