package snake

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
	"golang.org/x/crypto/ssh/terminal"
)

const millisecondsPerFrame = 16 // 16ms == 60fps

// Game of sneaky snakey goodness
type Game struct {
	level *Level

	// Score information for the current game
	score *Score

	// Size of the terminal
	size Size

	// Done status of the game
	done bool

	exit chan bool

	input chan termbox.Key
}

// NewGame starts a new game of Snake (bet you didn't guess that!)
func NewGame() *Game {
	game := &Game{}

	game.score = NewScore()
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
	game.size = Size{terminalWidth, terminalHeight}

	// FIXME: We might be able to call terminal.setSize() to control the size of our game, which would override resizing?

	// Calculate the width and height of the game, as well as taking into account power of two
	terminalWidth /= 2
	if terminalWidth%2 != 0 {
		terminalWidth--
	}
	if terminalHeight%2 != 0 {
		terminalHeight--
	}

	// Adjust height for UI elements
	// NOTE: Hyper works with -1, but Terminal requires -2
	terminalHeight--
	terminalHeight--

	// Create the level, which creates the snake, apple etc.
	game.level = NewLevel(game, Size{terminalWidth, terminalHeight}) // Divide by two to account for display character width

	// Offset the level to make room for UI elements
	// game.level.offset = Position{0, 1}

	return game
}

// IncrementScore will increase the score of the current game
func (game *Game) IncrementScore() {
	// log.Println("Incrementing game score from", game.score, "to", game.score+1)

	game.score.IncrementScore()
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

// Restart the game
func (game *Game) Restart() {
	// Reset the game score
	game.score = NewScore()

	// Recreate the level
	game.level = NewLevel(game, game.level.size)

	// Clear the terminal screen to avoid artefacting
	clearScreen()
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

			// Keep track of the frame start time
			frameStartTime := time.Now()

			// Clear the terminal screen
			clearScreen()

			// Update the level
			game.level.Update()

			// Draw the scores
			scoreString := "SCORE: " + strconv.Itoa(game.score.GetScore())
			if game.score.GetHighscore() > 0 {
				scoreString += " (HIGHSCORE: " + strconv.Itoa(game.score.GetHighscore()) + ")"
			}
			scoreString = CenterAlignString(scoreString, game.size.Width-len(scoreString))
			scoreStringLength := len(scoreString)
			paddedScoreString := scoreString
			if scoreStringLength < game.size.Width {
				for i := 0; i < game.size.Width-scoreStringLength; i++ {
					paddedScoreString += " "
				}
			}
			scoreString = "\033[47;30m" + paddedScoreString + "\033[0m"
			fmt.Println(scoreString)

			// Draw the level
			fmt.Println(game.level.Render())
			// fmt.Printf(game.level.Render())

			// Artificially delay the event loop to constrain the frames per second
			time.Sleep(frameStartTime.Add(time.Millisecond * millisecondsPerFrame).Sub(time.Now()))
		}
	}
}

// CenterAlignString will align the input string in
// the middle, based on the supplied width
func CenterAlignString(input string, width int) string {
	divider := width / 2
	return strings.Repeat(" ", divider) + input + strings.Repeat(" ", divider)
}
