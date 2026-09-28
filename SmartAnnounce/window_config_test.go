package main

import "testing"

func TestDefaultWindowLayoutFitsTheWorkbench(t *testing.T) {
	layout := defaultWindowLayout()

	if layout.Width != 1280 {
		t.Fatalf("expected width 1280, got %d", layout.Width)
	}

	if layout.Height != 820 {
		t.Fatalf("expected height 820, got %d", layout.Height)
	}

	if layout.MinWidth != 980 {
		t.Fatalf("expected min width 980, got %d", layout.MinWidth)
	}

	if layout.MinHeight != 680 {
		t.Fatalf("expected min height 680, got %d", layout.MinHeight)
	}
}
