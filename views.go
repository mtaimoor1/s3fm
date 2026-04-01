package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatSize returns a human-readable file size string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Color palette
var (
	purple    = lipgloss.Color("#7C3AED")
	violet    = lipgloss.Color("#A78BFA")
	slate     = lipgloss.Color("#94A3B8")
	darkSlate = lipgloss.Color("#475569")
	white     = lipgloss.Color("#F8FAFC")
	dimWhite  = lipgloss.Color("#CBD5E1")
	green     = lipgloss.Color("#34D399")
	amber     = lipgloss.Color("#FBBF24")
	red       = lipgloss.Color("#F87171")
)

// Styles
var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(amber)

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
	fileIcon        = lipgloss.NewStyle().Foreground(slate).Render("\U0001F4C4 ")
	bucketIcon      = lipgloss.NewStyle().Foreground(green).Render("\U0001F4E6 ")
	selectedPointer = lipgloss.NewStyle().Foreground(purple).Bold(true).Render("\u25B8 ")

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

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(darkSlate).
		Width(w - 2)

	// --- Header ---
	headerContent := m.renderHeader(w - 4) // account for box border + padding
	headerBox := boxStyle.Render(headerContent)

	// --- List content ---
	var listContent string
	if m.searching {
		listContent = m.renderSearchContent(vpHeight)
	} else if m.state == bucketList {
		listContent = m.renderBucketContent(vpHeight)
	} else {
		listContent = m.renderFileContent(vpHeight)
	}

	listBox := boxStyle.Render(listContent)

	// --- Footer ---
	footer := m.renderStatusBar(w)

	// --- Compose full layout ---
	view := lipgloss.JoinVertical(lipgloss.Left, headerBox, listBox, footer)

	// --- Help overlay ---
	if m.showHelp {
		view = m.renderHelpOverlay(w)
	}

	return view
}

// renderHeader renders the title bar and breadcrumb inside the header box.
func (m model) renderHeader(w int) string {
	// ASCII art logo
	asciiLogo := []string{
		"  ____  _____  ______ __  __ ",
		" / ___||___ / |  ____|  \\/  |",
		" \\___ \\  |_ \\ | |__  | \\  / |",
		"  ___) |___) ||  __| | |\\/| |",
		" |____/|____/ |_|    |_|  |_|",
	}

	var logoLines []string
	for _, line := range asciiLogo {
		logoLines = append(logoLines, logoStyle.Render(line))
	}
	logo := strings.Join(logoLines, "\n")

	// Info panel right-aligned
	labelStyle := lipgloss.NewStyle().Foreground(violet).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(dimWhite)

	regionLine := labelStyle.Render("Region  ") + valueStyle.Render(m.region)
	profileLine := labelStyle.Render("Profile ") + valueStyle.Render(m.profile)

	viewLabel := "Buckets"
	if m.state == fileList {
		viewLabel = m.currentBucket
	}
	viewLine := labelStyle.Render("View    ") + valueStyle.Render(viewLabel)

	infoBlock := lipgloss.JoinVertical(lipgloss.Left,
		regionLine,
		profileLine,
		viewLine,
	)

	// Join logo left, info right-aligned with gap fill
	logoWidth := lipgloss.Width(logo)
	infoWidth := lipgloss.Width(infoBlock)
	gap := w - logoWidth - infoWidth
	if gap < 2 {
		gap = 2
	}
	headerTop := lipgloss.JoinHorizontal(lipgloss.Center, logo, strings.Repeat(" ", gap), infoBlock)

	// Breadcrumb
	divider := lipgloss.NewStyle().Foreground(darkSlate).Render(strings.Repeat("\u2500", w))
	var crumb string
	if m.state == bucketList {
		label := breadcrumbActiveStyle.Render(" Buckets")
		cnt := countStyle.Render(fmt.Sprintf("  %d items", len(m.buckets)))
		crumb = label + cnt
	} else {
		parts := []string{}
		if m.startBucket == "" {
			parts = append(parts, breadcrumbStyle.Render(" Buckets"))
			parts = append(parts, separatorStyle.Render(" \u203A "))
		} else {
			parts = append(parts, breadcrumbStyle.Render(" "))
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

	return headerTop + "\n" + divider + "\n" + crumb
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

// Grid column widths
const (
	sizeColWidth = 10
	dateColWidth = 18
)

// renderGridHeader renders the column header row and divider for the file grid.
func (m model) renderGridHeader(innerW int) string {
	headingStyle := lipgloss.NewStyle().Foreground(violet).Bold(true)
	divStyle := lipgloss.NewStyle().Foreground(darkSlate)

	nameW := innerW - sizeColWidth - dateColWidth - 4 // 4 for spacing between cols
	if nameW < 10 {
		nameW = 10
	}

	nameCol := headingStyle.Width(nameW).Render("      Name")
	sizeCol := headingStyle.Width(sizeColWidth).Align(lipgloss.Right).Render("Size")
	dateCol := headingStyle.Width(dateColWidth).Align(lipgloss.Right).Render("Modified")

	header := nameCol + "  " + sizeCol + "  " + dateCol
	divider := divStyle.Render(strings.Repeat("\u2500", innerW))

	return header + "\n" + divider
}

// renderFileRow renders a single file/folder row with columns aligned to the grid.
func (m model) renderFileRow(f fileItem, selected bool, innerW int) string {
	icon := fileIcon
	if f.isDir {
		icon = folderIcon
	}

	var pointer string
	if selected {
		pointer = selectedPointer
	} else {
		pointer = "  "
	}

	nameW := innerW - sizeColWidth - dateColWidth - 4
	if nameW < 10 {
		nameW = 10
	}

	// Truncate name if too long (account for pointer + icon width ~4-5 chars)
	displayName := f.name
	maxName := nameW - 6
	if maxName < 4 {
		maxName = 4
	}
	if len(displayName) > maxName {
		displayName = displayName[:maxName-1] + "\u2026"
	}

	nameStr := fmt.Sprintf("%s%s%s", pointer, icon, displayName)

	var sizeStr, dateStr string
	metaStyle := lipgloss.NewStyle().Foreground(darkSlate)
	if !f.isDir {
		sizeStr = metaStyle.Width(sizeColWidth).Align(lipgloss.Right).Render(formatSize(f.size))
		dateStr = metaStyle.Width(dateColWidth).Align(lipgloss.Right).Render(f.lastModified.Format("Jan 02 2006 15:04"))
	} else {
		sizeStr = metaStyle.Width(sizeColWidth).Align(lipgloss.Right).Render("\u2500")
		dateStr = metaStyle.Width(dateColWidth).Align(lipgloss.Right).Render("\u2500")
	}

	// Pad name column to fixed width
	nameRenderedWidth := lipgloss.Width(nameStr)
	namePad := nameW - nameRenderedWidth
	if namePad < 0 {
		namePad = 0
	}

	row := nameStr + strings.Repeat(" ", namePad) + "  " + sizeStr + "  " + dateStr
	if selected {
		return selectedItemStyle.Render(row)
	}
	return itemStyle.Render(row)
}

// renderFileContent returns the file grid with header, padded to viewport height.
func (m model) renderFileContent(vpHeight int) string {
	w := m.width
	if w == 0 {
		w = 80
	}
	innerW := w - 4

	if len(m.files) == 0 {
		header := m.renderGridHeader(innerW)
		dh := vpHeight - 2
		if dh < 1 {
			dh = 1
		}
		return header + "\n" + padToHeight(emptyStyle.Render("This directory is empty."), dh)
	}

	header := m.renderGridHeader(innerW)
	dataHeight := vpHeight - 2 // subtract header + divider
	if dataHeight < 1 {
		dataHeight = 1
	}

	var lines []string
	start := m.yOffset
	end := start + dataHeight
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		lines = append(lines, m.renderFileRow(m.files[i], m.cursor == i, innerW))
	}

	return header + "\n" + padToHeight(strings.Join(lines, "\n"), dataHeight)
}

// renderSearchContent returns the search results padded to viewport height.
func (m model) renderSearchContent(vpHeight int) string {
	w := m.width
	if w == 0 {
		w = 80
	}
	innerW := w - 4
	isFileView := m.state == fileList

	// For file view, show grid header
	var headerStr string
	dataHeight := vpHeight
	if isFileView {
		headerStr = m.renderGridHeader(innerW)
		dataHeight = vpHeight - 2
	}

	if len(m.searchMatches) == 0 {
		empty := padToHeight(emptyStyle.Render("No matches found."), dataHeight)
		if headerStr != "" {
			return headerStr + "\n" + empty
		}
		return padToHeight(emptyStyle.Render("No matches found."), vpHeight)
	}

	var lines []string
	start := 0
	if m.searchCursor >= dataHeight {
		start = m.searchCursor - dataHeight + 1
	}
	end := start + dataHeight
	if end > len(m.searchMatches) {
		end = len(m.searchMatches)
	}

	for i := start; i < end; i++ {
		realIdx := m.searchMatches[i]
		selected := i == m.searchCursor

		if isFileView {
			lines = append(lines, m.renderFileRow(m.files[realIdx], selected, innerW))
		} else {
			name := m.buckets[realIdx]
			if selected {
				lines = append(lines, selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, bucketIcon, name)))
			} else {
				lines = append(lines, itemStyle.Render(fmt.Sprintf("  %s%s", bucketIcon, name)))
			}
		}
	}

	content := padToHeight(strings.Join(lines, "\n"), dataHeight)
	if headerStr != "" {
		return headerStr + "\n" + content
	}
	return content
}

// renderStatusBar renders the full-width footer bar pinned to the bottom.
func (m model) renderStatusBar(w int) string {
	barStyle := lipgloss.NewStyle().
		Foreground(slate).
		Width(w).
		Padding(0, 1)

	var left string

	if m.searching {
		prompt := lipgloss.NewStyle().Foreground(violet).Bold(true).Render("/")
		query := lipgloss.NewStyle().Foreground(white).Render(m.searchQuery)
		cursor := lipgloss.NewStyle().Foreground(violet).Bold(true).Render("\u2588")
		matchInfo := ""
		if m.searchQuery != "" {
			matchInfo = lipgloss.NewStyle().Foreground(darkSlate).
				Render(fmt.Sprintf("  %d match(es)", len(m.searchMatches)))
		}
		left = prompt + query + cursor + matchInfo
	} else if m.statusMsg != "" {
		left = lipgloss.NewStyle().Foreground(green).Bold(true).Render(m.statusMsg)
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
		sep := lipgloss.NewStyle().Foreground(darkSlate).Render(" \u2502 ")
		for _, k := range keys {
			hint := lipgloss.NewStyle().Foreground(violet).Bold(true).Render(k.key) +
				lipgloss.NewStyle().Foreground(slate).Render(" "+k.desc)
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
		visRows := m.visibleRows()
		if m.yOffset > 0 || m.yOffset+visRows < listLen {
			pct := 0
			if listLen > 1 {
				pct = m.cursor * 100 / (listLen - 1)
			}
			pos += fmt.Sprintf(" %d%%", pct)
		}
		right = lipgloss.NewStyle().Foreground(slate).Render(pos)
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
