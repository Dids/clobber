package snake

import (
	"fmt"
	"log"
	"time"

	"github.com/nsf/termbox-go"
	"golang.org/x/crypto/ssh/terminal"
)

const simulateMilliseconds = 100

// Game of sneaky snakey goodness
type Game struct {
	level *Level

	// Score of the current game
	score int // TODO: Add current and previous "highscore"

	// Done status of the game
	done bool

	exit chan bool

	input chan termbox.Key
}

// NewGame starts a new game of Snake (bet you didn't guess that!)
func NewGame() *Game {
	game := &Game{}

	game.score = 0
	game.done = false
	game.exit = make(chan bool)
	game.input = make(chan termbox.Key)

	// // Add support for graceful shutdown with CTRL-C
	// c := make(chan os.Signal)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// go func() {
	// 	<-c
	// 	game.done = true
	// 	game.exit <- true
	// }()

	// Get the terminal size
	terminalWidth, terminalHeight, err := terminal.GetSize(0)
	if err != nil {
		log.Fatal(err)
	}

	// FIXME: We might be able to call terminal.setSize() to control the size of our game, which would override resizing?

	// Calculate the width and height of the game, as well as taking into account power of two
	terminalWidth /= 2
	if terminalWidth%2 != 0 {
		terminalWidth--
	}
	if terminalHeight%2 != 0 {
		terminalHeight--
	}

	// Create the level, which creates the snake, apple etc.
	game.level = NewLevel(game, Size{terminalWidth, terminalHeight}) // Divide by two to account for display character width

	// Offset the level to make room for UI elements
	// game.level.offset = Position{0, 1}

	return game
}

// IncrementScore will increase the score of the current game
func (game *Game) IncrementScore() {
	// log.Println("Incrementing game score from", game.score, "to", game.score+1)

	game.score++
}

// Start will run the update loop in a goroutine
func (game *Game) Start() {
	log.Println("Game starting")

	// Initialize termbox
	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	// Start listening for keyboard input
	go ListenKeyboard(game.input)

	go game.update()

	// Block until an exit signal is received
	<-game.exit

	log.Println("Game is ending")
}

// Update loop for the game
func (game *Game) update() {
	for {
		select {
		case key := <-game.input:
			direction := GetInputDirection(key)
			// log.Println("Direction:", direction)
			if !direction.Zero() {
				game.level.snake.UpdateDirection(direction)
			} else if key == termbox.KeyEsc {
				log.Println("Escape pressed, exiting")
				game.done = true
				game.exit <- true
				return
			}
		default:
			// Quit the game if it's done
			if game.done {
				log.Println("Game done, exiting")
				return
			}

			// Clear the terminal screen
			clearScreen()

			// Draw the score
			fmt.Println("SCORE:", game.score)

			// Draw the level
			fmt.Println(game.level.Render())
			// fmt.Printf(game.level.Render())

			// Artificially delay the render/update loop
			time.Sleep(time.Millisecond * simulateMilliseconds)

			// Update the level
			game.level.Update()
		}
	}
}
