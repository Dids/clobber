package snake

import (
	"math/rand"
	"time"
)

const updateIntervalMs = 100

// Level of awesomeness?
type Level struct {
	// Game is the current game instance
	game *Game

	// Size of the level
	size Size

	lastUpdate time.Time

	snake *Snake
	apple *Apple
}

// NewLevel creates a new level of a certain size,
// also spawning in the snake and an apple
func NewLevel(game *Game, size Size) *Level {
	// log.Println("Creating a new level of size", size)

	level := &Level{}
	level.game = game
	level.size = size

	level.lastUpdate = time.Now()

	// Create a snake at roughly the center of the level
	level.snake = NewSnake(Position{level.size.Width / 2, level.size.Height / 2}, Direction{0, 1})

	// Create the initial apple
	level.apple = &Apple{level.GetRandomPosition()}

	return level
}

// EatApple will destroy the current apple, increment the snake size and score,
// finally spawning a new apple
func (level *Level) EatApple() {
	// log.Println("Apple was eaten")

	// Increment the size of the snake
	level.snake.IncrementSize()

	// Increment the game score
	level.game.score.IncrementScore()

	// Move the apple to a new location
	level.apple.position = level.GetRandomPosition()
}

// GetRandomPosition will return a randomize Position,
// constrained to the size of the current level
func (level *Level) GetRandomPosition() Position {
	x := rand.Intn(level.size.Width-2) + 1
	y := rand.Intn(level.size.Height-2) + 1
	// log.Println("Generated a random position at", Position{x, y})
	return Position{x, y}
}

// Render returns a string representation of the level
func (level *Level) Render() string {
	result := ""

	for y := 0; y < level.size.Height; y++ {
		for x := 0; x < level.size.Width; x++ {
			if level.snake.CheckHitbox(Position{x, y}) {
				result += "\033[42m  \033[0m"
			} else if level.apple.position.Equals(x, y) {
				result += "\033[41m  \033[0m"
			} else if level.IsWall((Position{x, y})) {
				result += "\033[47m  \033[0m"
			} else {
				result += "\033[40m  \033[0m"
			}
		}

		if y < level.size.Height-1 {
			result += "\n"
		}
	}

	return result
}

// IsWall will return true if the supplied position is a wall or a corner
func (level *Level) IsWall(pos Position) bool {
	if pos.X == 0 || pos.Y == 0 {
		return true
	}
	if pos.X == level.size.Width-1 || pos.Y == level.size.Height-1 {
		return true
	}
	return false
}

// Update keeps things moving
func (level *Level) Update() {
	// Constrain the level update loop to run much slower than the game loop,
	// but also increase the speed over time, based on the score
	if level.lastUpdate.Add(time.Millisecond*time.Duration(updateIntervalMs-(level.game.score.GetScore()/2))).Sub(time.Now()) > 0 {
		return
	}
	level.lastUpdate = time.Now()

	// Keep moving the snake
	level.snake.Move()

	// Keep checking if the snake collides with the apple,
	// then call EatApple() if it does
	if level.snake.CheckHitbox(level.apple.position) {
		level.EatApple()
	}

	// Check if colliding with a wall, then kill the snake
	if level.IsWall(level.snake.GetHead()) {
		level.snake.dead = true
	}

	// Check if the snake collides with itself, then start over
	if level.snake.dead {
		level.game.Restart()
		return
	}
}
