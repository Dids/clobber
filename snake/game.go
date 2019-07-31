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

	input  chan termbox.Key
	resize chan Size
}

// NewGame starts a new game of Snake (bet you didn't guess that!)
func NewGame() *Game {
	game := &Game{}

	game.score = NewScore()
	game.done = false
	game.exit = make(chan bool)
	game.input = make(chan termbox.Key)
	game.resize = make(chan Size)

	// // Add support for graceful shutdown with CTRL-C
	// c := make(chan os.Signal)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// go func() {
	// 	<-c
	// 	game.done = true
	// 	game.exit <- true
	// }()

	game.Resize()

	return game
}

// IncrementScore will increase the score of the current game
func (game *Game) IncrementScore() {
	game.score.IncrementScore()
}

// Start will run the update loop in a goroutine
func (game *Game) Start() {
	// Initialize termbox
	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	// Start listening for events
	go ListenEvents(game, game.input, game.resize)

	go game.update()

	// Block until an exit signal is received
	<-game.exit
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

// Resize adjusts the size of the game and level
func (game *Game) Resize() {
	// Get the terminal size
	terminalWidth, terminalHeight, err := terminal.GetSize(0)
	if err != nil {
		log.Fatal(err)
	}

	game.size = Size{terminalWidth, terminalHeight}

	levelWidth := game.size.Width
	levelHeight := game.size.Height

	// Calculate the width and height of the game, as well as taking into account power of two
	levelWidth /= 2
	if levelWidth%2 != 0 {
		levelWidth--
	}
	if levelHeight%2 != 0 {
		levelHeight--
	}

	// TODO: This is likely a rounding issue of some sort?
	// Adjust height for UI elements
	// NOTE: Hyper works with -1, but Terminal requires -2
	levelHeight--
	levelHeight--

	// Adjust height for Clover build process at the bottom
	levelHeight--

	// Update the level size
	game.level = NewLevel(game, Size{levelWidth, levelHeight})
}

// Update loop for the game
func (game *Game) update() {
	for {
		select {
		case key := <-game.input:
			direction := GetInputDirection(key)
			if !direction.Zero() {
				game.level.snake.UpdateDirection(direction)
			}
		case <-game.resize:
			// Terminal resize event received, update accordingly
			game.Resize()
		default:
			// Quit the game if it's done
			if game.done {
				//log.Println("Game done, skipping update")
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
			scoreString = CenterAlignString(scoreString, (game.level.size.Width*2)-len(scoreString))
			scoreStringLength := len(scoreString)
			paddedScoreString := scoreString
			if scoreStringLength < (game.level.size.Width * 2) {
				for i := 0; i < (game.level.size.Width*2)-scoreStringLength; i++ {
					paddedScoreString += " "
				}
			}
			scoreString = "\033[47;30m" + paddedScoreString + "\033[0m"
			fmt.Println(scoreString)

			// Draw the level
			fmt.Println(game.level.Render())

			// Artificially delay the event loop to constrain the frames per second
			time.Sleep(frameStartTime.Add(time.Millisecond * millisecondsPerFrame).Sub(time.Now()))

			// Fix resizing artefacts by flushing terminal back buffer
			termbox.Flush()
		}
	}
}

// CenterAlignString will align the input string in
// the middle, based on the supplied width
func CenterAlignString(input string, width int) string {
	divider := width / 2
	return strings.Repeat(" ", divider) + input + strings.Repeat(" ", divider)
}
