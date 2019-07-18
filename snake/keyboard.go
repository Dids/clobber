package snake

import (
	"log"

	"github.com/nsf/termbox-go"
)

// ListenKeyboard will start listening for keyboard events,
// then emit them on the supplied channel as keys
func ListenKeyboard(event chan termbox.Key) {
	// TODO: What does this do?
	termbox.SetInputMode(termbox.InputEsc)

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			event <- ev.Key // GetInputDirection(ev.Key)
			// switch ev.Key {
			// case termbox.KeyArrowLeft:
			// 	event <- keyboardEvent{eventType: MOVE, key: ev.Key}
			// case termbox.KeyArrowDown:
			// 	event <- keyboardEvent{eventType: MOVE, key: ev.Key}
			// case termbox.KeyArrowRight:
			// 	event <- keyboardEvent{eventType: MOVE, key: ev.Key}
			// case termbox.KeyArrowUp:
			// 	event <- keyboardEvent{eventType: MOVE, key: ev.Key}
			// case termbox.KeyEsc:
			// 	event <- keyboardEvent{eventType: END, key: ev.Key}
			// default:
			// 	if ev.Ch == 'r' {
			// 		event <- keyboardEvent{eventType: RETRY, key: ev.Key}
			// 	}
			// }
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
