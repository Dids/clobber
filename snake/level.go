package snake

import (
	"log"
	"math/rand"
)

// Level of awesomeness?
type Level struct {
	// Game is the current game instance
	game *Game

	// Size of the level
	size Size

	// Offset of the level
	// offset Position

	snake *Snake
	apple *Apple
}

// NewLevel creates a new level of a certain size,
// also spawning in the snake and an apple
func NewLevel(game *Game, size Size) *Level {
	log.Println("Creating a new level of size", size)

	level := &Level{}

	level.game = game
	level.size = size

	// Create a snake at roughly the center of the level
	level.snake = NewSnake(Position{size.Width / 2, size.Height / 2}, Direction{0, 1})

	// Create the initial apple
	level.apple = &Apple{level.GetRandomPosition()}

	// Eat the apple
	// level.EatApple()

	return level
}

// EatApple will destroy the current apple, increment the snake size and score,
// finally spawning a new apple
func (level *Level) EatApple() {
	// log.Println("Apple was eaten")

	// Increment the size of the snake
	level.snake.IncrementSize()

	// Increment the game score
	level.game.score++

	// Move the apple to a new location
	level.apple.position = level.GetRandomPosition()
}

// GetRandomPosition will return a randomize Position,
// constrained to the size of the current level
func (level *Level) GetRandomPosition() Position {
	x := rand.Intn(level.size.Width)
	y := rand.Intn(level.size.Height)
	// log.Println("Generated a random position at", Position{x, y})
	return Position{x, y}
}

// Render returns a string representation of the level
func (level *Level) Render() string {
	result := ""

	for y := 0; y < level.size.Width; y++ {
		for x := 0; x < level.size.Width; x++ {
			if level.snake.CheckHitbox(Position{x, y}) {
				result += "\033[42m  \033[0m"
			} else if level.apple.position.Equals(x, y) {
				result += "\033[41m  \033[0m"
			} else {
				result += "\033[97m  \033[0m"
			}
		}

		if y < level.size.Height-1 {
			result += "\n"
		}
	}

	return result
}

// Update keeps things moving
func (level *Level) Update() {
	// FIXME: The snake should move at a fixed interval, while the game loop (esp. input) should run much faster

	// Keep moving the snake
	level.snake.Move()

	// Check if the snake collides with itself, then start over
	if level.snake.dead {
		// TODO: Actually implement a reset mechanic
		// log.Println("Snake collided with self and died")
		level.game.done = true
		level.game.exit <- true
		return
	}

	// Keep checking if the snake collides with the apple,
	// then call EatApple() if it does
	if level.snake.CheckHitbox(level.apple.position) {
		level.EatApple()
	}

	// TODO: If the snake it out of bounds, move it to the opposite end
	if level.snake.GetHead().X < 0 {
		// Left bounds, move to right side
	} else if level.snake.GetHead().X > level.size.Width {
		// Right bounds, move to left side
	} else if level.snake.GetHead().Y < 0 {
		// Bottom bounds, move to top side

	} else if level.snake.GetHead().Y > level.size.Height {
		// Top bounds, move to bottom side
	}
}
