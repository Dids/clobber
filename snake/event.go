package snake

import (
	"log"

	"github.com/nsf/termbox-go"
)

// ListenEvents will start listening for keyboard and resize events,
// then emit them on the supplied channels
func ListenEvents(inputEvent chan termbox.Key, resizeEvent chan Size) {
	termbox.SetInputMode(termbox.InputEsc)
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			inputEvent <- ev.Key
		case termbox.EventResize:
			resizeEvent <- Size{ev.Width, ev.Height}
		case termbox.EventError:
			log.Fatal(ev.Err)
		}
	}
}

// GetInputDirection will convert a keyboard key to a Direction
func GetInputDirection(key termbox.Key) Direction {
	// log.Println("Get direction from input:", key)

	switch key {
	case termbox.KeyArrowLeft:
		return Direction{-1, 0}
	case termbox.KeyArrowDown:
		return Direction{0, 1}
	case termbox.KeyArrowRight:
		return Direction{+1, 0}
	case termbox.KeyArrowUp:
		return Direction{0, -1}
	default:
		return Direction{0, 0}
	}
}
