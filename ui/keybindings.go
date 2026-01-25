package ui

import (
	"github.com/gdamore/tcell/v2"
)

type KeyAction struct {
	name    string
	handler func()
}

type KeyBinding struct {
	action KeyAction
	keys   []tcell.Key // for special keys like arrows, pgdn, etc.
	runes  []rune      // for character keys
}

type KeyBindingManager struct {
	bindings map[tcell.Key]KeyAction // special key -> action mapping
	runeMap  map[rune]KeyAction      // rune -> action mapping
	pending  string                  // pending key sequence for multi-key bindings like 'gg'
}

func NewKeyBindingManager() *KeyBindingManager {
	return &KeyBindingManager{
		bindings: make(map[tcell.Key]KeyAction),
		runeMap:  make(map[rune]KeyAction),
		pending:  "",
	}
}

func (km *KeyBindingManager) RegisterKeyBinding(action KeyAction, keys []tcell.Key, runes []rune) {
	for _, key := range keys {
		km.bindings[key] = action
	}
	for _, r := range runes {
		km.runeMap[r] = action
	}
}

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

	r := event.Rune()

	if km.pending == "g" {
		km.pending = ""
		if r == 'g' {
			if action, ok := km.runeMap['G']; ok && action.name == "goStart" {
				action.handler()
				return true
			}
		}
		if action, ok := km.runeMap[r]; ok {
			action.handler()
			return true
		}
		return false
	}

	if r == 'g' {
		km.pending = "g"
		return true
	}

	if action, ok := km.runeMap[r]; ok {
		km.pending = ""
		action.handler()
		return true
	}

	km.pending = ""
	return false
}

func (km *KeyBindingManager) ResetPending() {
	km.pending = ""
}
