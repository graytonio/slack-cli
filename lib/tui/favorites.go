package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/graytonio/slack-cli/lib/config"
)

const maxFavorites = 9

// FavoritesModel manages the favorites overlay.
type FavoritesModel struct {
	items   []config.FavoriteChannel
	visible bool
	cursor  int
	width   int
	height  int
}

func NewFavoritesModel() FavoritesModel {
	cfg := config.GetConfig()
	items := make([]config.FavoriteChannel, len(cfg.FavoriteChannels))
	copy(items, cfg.FavoriteChannels)
	return FavoritesModel{
		items: items,
	}
}

// Update handles keys when the overlay is visible.
// Returns the updated model, an optional tea.Cmd, and an optional ChannelSelectedMsg
// if the user picked a favorite.
func (m FavoritesModel) Update(msg tea.Msg) (FavoritesModel, tea.Cmd, *ChannelSelectedMsg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Number keys 1-9: jump to slot
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			idx := int(key[0]-'0') - 1
			if idx < len(m.items) && m.items[idx].ID != "" {
				m.visible = false
				return m, nil, &ChannelSelectedMsg{
					ChannelID:   m.items[idx].ID,
					ChannelName: m.items[idx].Name,
				}
			}
			return m, nil, nil
		}

		switch key {
		case "esc":
			m.visible = false
			return m, nil, nil

		case "j", "down":
			if m.cursor < maxFavorites-1 {
				m.cursor++
			}
			return m, nil, nil

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil, nil

		case "enter":
			if m.cursor < len(m.items) && m.items[m.cursor].ID != "" {
				m.visible = false
				return m, nil, &ChannelSelectedMsg{
					ChannelID:   m.items[m.cursor].ID,
					ChannelName: m.items[m.cursor].Name,
				}
			}
			return m, nil, nil

		case "d", "backspace":
			if m.cursor < len(m.items) && m.items[m.cursor].ID != "" {
				m.items = append(m.items[:m.cursor], m.items[m.cursor+1:]...)
				if m.cursor >= len(m.items) && m.cursor > 0 {
					m.cursor--
				}
				return m, m.persistCmd(), nil
			}
			return m, nil, nil

		case "J": // shift+j — move item down
			if m.cursor < len(m.items)-1 && m.cursor < maxFavorites-1 {
				m.items[m.cursor], m.items[m.cursor+1] = m.items[m.cursor+1], m.items[m.cursor]
				m.cursor++
				return m, m.persistCmd(), nil
			}
			return m, nil, nil

		case "K": // shift+k — move item up
			if m.cursor > 0 && m.cursor < len(m.items) {
				m.items[m.cursor], m.items[m.cursor-1] = m.items[m.cursor-1], m.items[m.cursor]
				m.cursor--
				return m, m.persistCmd(), nil
			}
			return m, nil, nil
		}
	}
	return m, nil, nil
}

func (m FavoritesModel) View() string {
	var sb strings.Builder

	sb.WriteString(favTitleStyle.Render("★ Favorites") + "\n\n")

	for i := 0; i < maxFavorites; i++ {
		var line string
		if i < len(m.items) && m.items[i].ID != "" {
			line = fmt.Sprintf(" %d  #%s", i+1, m.items[i].Name)
			if i == m.cursor {
				line = favSelectedStyle.Render(line)
			} else {
				line = favItemStyle.Render(line)
			}
		} else {
			line = fmt.Sprintf(" %d  ---", i+1)
			if i == m.cursor {
				line = favSelectedStyle.Render(line)
			} else {
				line = favEmptyStyle.Render(line)
			}
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n" + favHelpStyle.Render("1-9 jump  enter select  d delete  J/K reorder  esc close"))

	return favOverlayStyle.Render(sb.String())
}

// Add adds a channel to the next available favorite slot.
// Returns false if already at max capacity.
func (m *FavoritesModel) Add(id, name string) bool {
	// Don't add duplicates
	for _, f := range m.items {
		if f.ID == id {
			return false
		}
	}
	if len(m.items) >= maxFavorites {
		return false
	}
	m.items = append(m.items, config.FavoriteChannel{ID: id, Name: name})
	return true
}

// Remove removes a channel from favorites by ID.
// Returns true if found and removed.
func (m *FavoritesModel) Remove(id string) bool {
	for i, f := range m.items {
		if f.ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return true
		}
	}
	return false
}

// SlotFor returns the 1-based slot number for a channel, or 0 if not favorited.
func (m FavoritesModel) SlotFor(channelID string) int {
	for i, f := range m.items {
		if f.ID == channelID {
			return i + 1
		}
	}
	return 0
}

// IsFavorite returns true if the channel is in the favorites list.
func (m FavoritesModel) IsFavorite(channelID string) bool {
	return m.SlotFor(channelID) > 0
}

// GetSlot returns the favorite at the given 0-based index, or nil if empty/out of range.
func (m FavoritesModel) GetSlot(idx int) *config.FavoriteChannel {
	if idx < 0 || idx >= len(m.items) || m.items[idx].ID == "" {
		return nil
	}
	return &m.items[idx]
}

func (m FavoritesModel) persistCmd() tea.Cmd {
	items := make([]config.FavoriteChannel, len(m.items))
	copy(items, m.items)
	return func() tea.Msg {
		config.SaveFavorites(items)
		return FavoritesSavedMsg{}
	}
}
