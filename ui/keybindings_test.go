package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestKeyBindingManager(t *testing.T) {
	km := NewKeyBindingManager()

	// Test single key binding
	handledSpace := false
	km.RegisterKeyBinding(
		KeyAction{
			name:    "toggle",
			handler: func() { handledSpace = true },
		},
		[]tcell.Key{},
		[]rune{' '},
	)

	event := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	if !km.HandleKey(event) {
		t.Errorf("Expected space key to be handled")
	}
	if !handledSpace {
		t.Errorf("Expected handler to be called")
	}

	// Test 'g' prefix sequence
	goStartCalled := false
	km.RegisterKeyBinding(
		KeyAction{
			name:    "goStart",
			handler: func() { goStartCalled = true },
		},
		[]tcell.Key{},
		[]rune{'G'},
	)

	// First 'g' should be pending
	event1 := tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone)
	if !km.HandleKey(event1) {
		t.Errorf("Expected first 'g' to be consumed")
	}
	if goStartCalled {
		t.Errorf("Handler should not be called yet")
	}

	// Second 'g' should trigger goStart
	event2 := tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone)
	if !km.HandleKey(event2) {
		t.Errorf("Expected second 'g' (gg sequence) to be handled")
	}
	if !goStartCalled {
		t.Errorf("Expected handler to be called for 'gg'")
	}
}

func TestKeyBindingManagerReset(t *testing.T) {
	km := NewKeyBindingManager()

	goStartCalled := false
	km.RegisterKeyBinding(
		KeyAction{
			name:    "goStart",
			handler: func() { goStartCalled = true },
		},
		[]tcell.Key{},
		[]rune{'G'},
	)

	// Press 'g'
	event1 := tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModNone)
	km.HandleKey(event1)

	// Press non-'g' key - should reset pending
	handleOtherCalled := false
	km.RegisterKeyBinding(
		KeyAction{
			name:    "other",
			handler: func() { handleOtherCalled = true },
		},
		[]tcell.Key{},
		[]rune{'h'},
	)

	event2 := tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone)
	if !km.HandleKey(event2) {
		t.Errorf("Expected 'h' to be handled")
	}
	if !handleOtherCalled {
		t.Errorf("Expected 'h' handler to be called")
	}
	if goStartCalled {
		t.Errorf("goStart should not have been called")
	}
}
