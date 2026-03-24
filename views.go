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
)

// Styles
var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(violet).
			PaddingRight(1)

	titleBarStyle = lipgloss.NewStyle().
			Background(headerBg).
			Foreground(white).
			Bold(true).
			Padding(0, 1)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(slate).
			PaddingLeft(2)

	breadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(violet).
				Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(white).
				Bold(true).
				PaddingLeft(1).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(purple)

	folderIcon      = lipgloss.NewStyle().Foreground(amber).Render("\U0001F4C1 ")
	fileIcon        = lipgloss.NewStyle().Foreground(slate).Render("   ")
	bucketIcon      = lipgloss.NewStyle().Foreground(green).Render("\U0001F4E6 ")
	selectedPointer = lipgloss.NewStyle().Foreground(purple).Bold(true).Render("\u25B8 ")

	statusBarStyle = lipgloss.NewStyle().
			Foreground(darkSlate).
			PaddingLeft(2)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(violet).
			Bold(true)

	statusDescStyle = lipgloss.NewStyle().
			Foreground(darkSlate)

	scrollIndicatorStyle = lipgloss.NewStyle().
				Foreground(slate).
				PaddingLeft(2)

	countStyle = lipgloss.NewStyle().
			Foreground(slate).
			PaddingLeft(2)

	errorBoxStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red).
			Padding(1, 2)

	emptyStyle = lipgloss.NewStyle().
			Foreground(darkSlate).
			Italic(true).
			PaddingLeft(4)
)

const headerHeight = 4 // title bar + breadcrumb + blank line + blank
const footerHeight = 3 // blank + status bar + scroll info

func (m model) View() string {
	w := m.width
	if w == 0 {
		w = 80
	}

	if m.err != nil {
		errText := fmt.Sprintf("  Error: %v", m.err)
		box := errorBoxStyle.Width(w - 6).Render(errText)
		hint := lipgloss.NewStyle().Foreground(slate).PaddingLeft(2).Render("Press any key to quit.")
		return fmt.Sprintf("\n%s\n\n%s\n", box, hint)
	}

	viewportHeight := m.height - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	var b strings.Builder

	// --- Header ---
	if m.state == bucketList {
		m.renderBucketHeader(&b, w)
		m.renderBucketList(&b, viewportHeight)
	} else {
		m.renderFileHeader(&b, w)
		m.renderFileList(&b, viewportHeight)
	}

	// --- Footer ---
	m.renderFooter(&b, viewportHeight)

	view := b.String()

	// --- Help overlay ---
	if m.showHelp {
		view = m.renderHelpOverlay(w)
	}

	return view
}

func (m model) renderBucketHeader(b *strings.Builder, w int) {
	logo := logoStyle.Render("s3fm")
	region := lipgloss.NewStyle().Foreground(slate).Render(m.region)
	title := titleBarStyle.Width(w).Render(fmt.Sprintf("%s  %s", logo, region))
	b.WriteString(title + "\n")

	crumb := breadcrumbStyle.Render("Buckets")
	count := countStyle.Render(fmt.Sprintf("(%d items)", len(m.buckets)))
	b.WriteString(crumb + count + "\n\n")
}

func (m model) renderFileHeader(b *strings.Builder, w int) {
	logo := logoStyle.Render("s3fm")
	region := lipgloss.NewStyle().Foreground(slate).Render(m.region)
	title := titleBarStyle.Width(w).Render(fmt.Sprintf("%s  %s", logo, region))
	b.WriteString(title + "\n")

	// Breadcrumb: Buckets > bucket-name > path > parts
	parts := []string{}
	if m.startBucket == "" {
		parts = append(parts, breadcrumbStyle.Render("Buckets"))
		parts = append(parts, breadcrumbStyle.Render(" > "))
	}
	parts = append(parts, breadcrumbActiveStyle.Render(m.currentBucket))

	if m.currentPrefix != "" {
		segments := strings.Split(strings.TrimSuffix(m.currentPrefix, "/"), "/")
		for _, seg := range segments {
			parts = append(parts, breadcrumbStyle.Render(" > "))
			parts = append(parts, breadcrumbActiveStyle.Render(seg))
		}
	}

	crumb := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	count := countStyle.Render(fmt.Sprintf("(%d items)", len(m.files)))

	// Truncate if needed
	crumbLine := crumb + count
	_ = crumbLine
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, crumb, count) + "\n\n")
}

func (m model) renderBucketList(b *strings.Builder, viewportHeight int) {
	if len(m.buckets) == 0 {
		b.WriteString(emptyStyle.Render("No buckets found.") + "\n")
		return
	}

	start := m.yOffset
	end := start + viewportHeight
	if end > len(m.buckets) {
		end = len(m.buckets)
	}

	for i := start; i < end; i++ {
		name := m.buckets[i]
		if m.cursor == i {
			line := selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, bucketIcon, name))
			b.WriteString(line + "\n")
		} else {
			line := itemStyle.Render(fmt.Sprintf("  %s%s", bucketIcon, name))
			b.WriteString(line + "\n")
		}
	}
}

func (m model) renderFileList(b *strings.Builder, viewportHeight int) {
	if len(m.files) == 0 {
		b.WriteString(emptyStyle.Render("This directory is empty.") + "\n")
		return
	}

	start := m.yOffset
	end := start + viewportHeight
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
			line := selectedItemStyle.Render(fmt.Sprintf("%s%s%s", selectedPointer, icon, name))
			b.WriteString(line + "\n")
		} else {
			line := itemStyle.Render(fmt.Sprintf("  %s%s", icon, name))
			b.WriteString(line + "\n")
		}
	}
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

func (m model) renderFooter(b *strings.Builder, viewportHeight int) {
	listLen := len(m.buckets)
	if m.state == fileList {
		listLen = len(m.files)
	}

	b.WriteString("\n")

	// Status message (e.g. "Copied: s3://...") or key hints
	if m.statusMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(green).Bold(true).PaddingLeft(2)
		b.WriteString(msgStyle.Render(m.statusMsg) + "\n")
	} else {
		keys := []struct{ key, desc string }{
			{"j/k", "navigate"},
			{"enter", "open"},
			{"esc", "back"},
			{"yy", "copy path"},
			{"G/g", "top/bottom"},
			{"q", "quit"},
			{"?", "help"},
		}
		var hints []string
		for _, k := range keys {
			hints = append(hints, statusKeyStyle.Render(k.key)+" "+statusDescStyle.Render(k.desc))
		}
		b.WriteString(statusBarStyle.Render(strings.Join(hints, "  "+statusDescStyle.Render("|")+"  ")) + "\n")
	}

	// Scroll position
	if listLen > 0 {
		pos := fmt.Sprintf("%d/%d", m.cursor+1, listLen)
		scrollInfo := scrollIndicatorStyle.Render(pos)

		if m.yOffset > 0 || m.yOffset+viewportHeight < listLen {
			pct := 0
			if listLen > 1 {
				pct = m.cursor * 100 / (listLen - 1)
			}
			scrollInfo += statusDescStyle.Render(fmt.Sprintf("  %d%%", pct))
		}
		b.WriteString(scrollInfo)
	}
}
