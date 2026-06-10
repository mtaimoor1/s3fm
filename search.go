package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSearchInput processes keystrokes while in search mode.
func (m model) handleSearchInput(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.search = searchState{}
		return m, nil

	case "enter":
		// Apply the current matches as a persistent filter and exit search mode
		if len(m.search.matches) > 0 {
			selectedPos := m.search.cursor
			m = m.applyFilter()
			m.cursor = selectedPos
			m = m.clampScroll()
		}
		m.search = searchState{}
		return m, nil

	case "backspace":
		if len(m.search.query) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.search.query)
			m.search.query = m.search.query[:len(m.search.query)-size]
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
			m.search.query += b.String()
			m = m.recomputeSearchMatches()
		}
		return m, nil

	case "up", "ctrl+p":
		if m.search.cursor > 0 {
			m.search.cursor--
		}
		return m, nil

	case "down", "ctrl+n":
		if m.search.cursor < len(m.search.matches)-1 {
			m.search.cursor++
		}
		return m, nil

	default:
		if utf8.RuneCountInString(key) == 1 {
			r, _ := utf8.DecodeRuneInString(key)
			if unicode.IsPrint(r) {
				m.search.query += key
				m = m.recomputeSearchMatches()
			}
		}
		return m, nil
	}
}

// recomputeSearchMatches filters the current list by the search query.
func (m model) recomputeSearchMatches() model {
	m.search.matches = nil
	m.search.cursor = 0
	query := strings.ToLower(m.search.query)
	if m.state == bucketList {
		for i, item := range m.buckets {
			if query == "" || strings.Contains(strings.ToLower(item), query) {
				m.search.matches = append(m.search.matches, i)
			}
		}
	} else {
		for i, item := range m.files {
			if query == "" || strings.Contains(strings.ToLower(item.name), query) {
				m.search.matches = append(m.search.matches, i)
			}
		}
	}
	return m
}

// applyFilter replaces the current list with only the search-matched items.
func (m model) applyFilter() model {
	if m.state == bucketList {
		unfiltered := make([]string, len(m.buckets))
		copy(unfiltered, m.buckets)
		m.filter.unfilteredBuckets = unfiltered
		filtered := make([]string, 0, len(m.search.matches))
		for _, idx := range m.search.matches {
			filtered = append(filtered, m.buckets[idx])
		}
		m.buckets = filtered
	} else {
		unfiltered := make([]fileItem, len(m.files))
		copy(unfiltered, m.files)
		m.filter.unfilteredFiles = unfiltered
		filtered := make([]fileItem, 0, len(m.search.matches))
		for _, idx := range m.search.matches {
			filtered = append(filtered, m.files[idx])
		}
		m.files = filtered
	}
	m.filter.active = true
	m.yOffset = 0
	return m
}

// clearFilter restores the full unfiltered list.
func (m model) clearFilter() model {
	if m.state == bucketList && m.filter.unfilteredBuckets != nil {
		m.buckets = m.filter.unfilteredBuckets
	} else if m.state == fileList && m.filter.unfilteredFiles != nil {
		m.files = m.filter.unfilteredFiles
	}
	m.filter = filterState{}
	m.cursor = 0
	m.yOffset = 0
	return m
}
