package ui

import (
	"github.com/gdamore/tcell/v2"
)

// KeyAction represents an action that can be triggered by keybindings
type KeyAction struct {
	name    string
	handler func()
}

// KeyBinding maps a set of keys to a single action
type KeyBinding struct {
	action KeyAction
	keys   []tcell.Key // for special keys like arrows, pgdn, etc.
	runes  []rune      // for character keys
}

// KeyBindingManager manages all keybindings and dispatches events
type KeyBindingManager struct {
	bindings map[tcell.Key]KeyAction // special key -> action mapping
	runeMap  map[rune]KeyAction      // rune -> action mapping
	pending  string                  // pending key sequence for multi-key bindings like 'gg'
}

// NewKeyBindingManager creates a new key binding manager
func NewKeyBindingManager() *KeyBindingManager {
	return &KeyBindingManager{
		bindings: make(map[tcell.Key]KeyAction),
		runeMap:  make(map[rune]KeyAction),
		pending:  "",
	}
}

// RegisterKeyBinding registers a single key binding
func (km *KeyBindingManager) RegisterKeyBinding(action KeyAction, keys []tcell.Key, runes []rune) {
	for _, key := range keys {
		km.bindings[key] = action
	}
	for _, r := range runes {
		km.runeMap[r] = action
	}
}

// HandleKey handles a keyboard event and returns true if it was consumed
func (km *KeyBindingManager) HandleKey(event *tcell.EventKey) bool {
	// Check for special keys first
	if event.Key() != tcell.KeyRune {
		if action, ok := km.bindings[event.Key()]; ok {
			km.pending = "" // reset pending sequence
			action.handler()
			return true
		}
		km.pending = "" // reset pending sequence on non-rune key
		return false
	}

	// Handle rune keys
	r := event.Rune()

	// Handle 'g' prefix for multi-key sequences
	if km.pending == "g" {
		km.pending = ""
		// 'gg' sequence - go to first page
		if r == 'g' {
			if action, ok := km.runeMap['G']; ok && action.name == "goStart" {
				action.handler()
				return true
			}
		}
		// Not a complete sequence, try current rune as standalone
		if action, ok := km.runeMap[r]; ok {
			action.handler()
			return true
		}
		return false
	}

	// Start potential sequence with 'g'
	if r == 'g' {
		km.pending = "g"
		return true
	}

	// Single character binding
	if action, ok := km.runeMap[r]; ok {
		km.pending = ""
		action.handler()
		return true
	}

	km.pending = ""
	return false
}

// ResetPending resets the pending key sequence
func (km *KeyBindingManager) ResetPending() {
	km.pending = ""
}
