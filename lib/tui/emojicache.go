package tui

import (
	"regexp"
	"strings"
	"sync"

	"github.com/kyokomi/emoji/v2"
)

var shortcodeRe = regexp.MustCompile(`:[a-zA-Z0-9_+-]+:`)

// EmojiCache resolves Slack :shortcode: syntax to Unicode characters.
// Standard emoji are resolved via a static mapping library; custom workspace
// emoji are resolved via the Slack API.
type EmojiCache struct {
	mu     sync.RWMutex
	custom map[string]string // workspace emoji from Slack API
}

func NewEmojiCache() *EmojiCache {
	return &EmojiCache{}
}

// SetCustom stores the workspace emoji map from Slack's emoji.list API.
func (c *EmojiCache) SetCustom(emojis map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.custom = emojis
}

// Replace converts :shortcode: sequences in text to their Unicode equivalents.
// Resolution order: custom alias → custom URL (styled fallback) → standard library → leave as-is.
func (c *EmojiCache) Replace(text string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return shortcodeRe.ReplaceAllStringFunc(text, func(match string) string {
		name := match[1 : len(match)-1] // strip surrounding colons

		// Check custom workspace emoji
		if c.custom != nil {
			if val, ok := c.custom[name]; ok {
				return c.resolveCustom(val, name)
			}
		}

		// Try standard emoji library
		candidate := emoji.Sprint(match)
		if candidate != match {
			return strings.TrimSpace(candidate)
		}

		// Leave as-is if unrecognized
		return match
	})
}

// resolveCustom handles a custom emoji value. Aliases start with "alias:"
// and point to another emoji name. URL values have no Unicode form so we
// render a styled fallback.
func (c *EmojiCache) resolveCustom(val, name string) string {
	if target, ok := strings.CutPrefix(val, "alias:"); ok {

		// Check if alias points to another custom emoji
		if c.custom != nil {
			if v2, ok := c.custom[target]; ok {
				return c.resolveCustom(v2, target)
			}
		}

		// Try standard emoji library for the alias target
		code := ":" + target + ":"
		candidate := emoji.Sprint(code)
		if candidate != code {
			return strings.TrimSpace(candidate)
		}
	}

	// Image URL or unresolvable alias — styled fallback
	return customEmojiStyle.Render("[:" + name + ":]")
}
