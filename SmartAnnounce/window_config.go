package main

type windowLayout struct {
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
}

func defaultWindowLayout() windowLayout {
	return windowLayout{
		Width:     1280,
		Height:    820,
		MinWidth:  980,
		MinHeight: 680,
	}
}
