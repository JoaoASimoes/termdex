package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Panels taller than the terminal are clipped and can be scrolled with the scroll keys.
// The scroll offset (m.scroll) resets whenever another key is pressed, so it always
// starts at the top of whatever is shown.

// uiState is written by View (which has a value receiver) and read by Update.
type uiState struct {
	maxScroll int // largest useful scroll offset of the panels drawn last
}

const scrollStep = 3

// availHeight is the height left for the panels' content, given the rendered footer.
// Above the panels: title, breadcrumb or blank, tabs or caption, blank; plus 2 border rows.
func (m tuiModel) availHeight(footer string) int {
	if m.height <= 0 {
		return 0 // unknown: do not clip
	}
	return max(6, m.height-6-lipgloss.Height(footer))
}

// clip cuts content to height lines at the current scroll offset, with an indicator line
// when there is more. keys names the scroll keys for the indicator.
func (m tuiModel) clip(content string, height int, keys string) string {
	if height <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= height {
		return content
	}
	window := height - 1 // the last line is the indicator
	maxOff := len(lines) - window
	if m.ui != nil && maxOff > m.ui.maxScroll {
		m.ui.maxScroll = maxOff
	}
	off := clamp(m.scroll, 0, maxOff)
	visible := lines[off : off+window]

	var arrows string
	switch {
	case off == 0:
		arrows = "↓ more"
	case off == maxOff:
		arrows = "↑ more"
	default:
		arrows = "↑↓ more"
	}
	if f := strings.Fields(keys); len(f) > 0 {
		keys = f[0] // only the main keys, so the line never wraps
	}
	indicator := labelStyle.Render(fmt.Sprintf("── %s · %s · %d%% ──", arrows, keys, 100*(off+window)/len(lines)))
	return strings.Join(visible, "\n") + "\n" + indicator
}

// scrollBy moves the scroll offset, staying within what was drawn.
func (m *tuiModel) scrollBy(delta int) {
	limit := 0
	if m.ui != nil {
		limit = m.ui.maxScroll
	}
	m.scroll = clamp(m.scroll+delta, 0, limit)
}
