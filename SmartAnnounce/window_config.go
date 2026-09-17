package main

type windowLayout struct {
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
}

func defaultWindowLayout() windowLayout {
	return windowLayout{
		Width:     1180,
		Height:    760,
		MinWidth:  980,
		MinHeight: 680,
	}
}
