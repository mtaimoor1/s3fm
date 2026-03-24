package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type initMsg struct {
	client  *s3Con
	buckets []string
	files   []string
	err     error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	const headerHeight = 4
	const footerHeight = 3
	viewportHeight := m.height - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

		switch msg.String() {
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
			if m.state == bucketList {
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
			} else if m.state == fileList {
				selected := m.files[m.cursor]
				if strings.HasSuffix(selected, "/") {
					newPrefix := m.currentPrefix + selected
					files, err := m.s3Client.listPrefix(m.currentBucket, newPrefix)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.files = files
					m.currentPrefix = newPrefix
					m.cursor = 0
					m.yOffset = 0
				}
			}
		case "esc", "backspace":
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
			m.cursor = 0
			m.yOffset = 0
		case "g":
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
