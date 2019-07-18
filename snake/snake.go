package snake

// Snake wants an Apple
type Snake struct {
	// Length of the body
	length int

	// Body of the snake (including head)
	body []Position

	// Direction of the snake
	direction Direction

	// Dead or alive
	dead bool
}

// NewSnake spawns a hissing creature
func NewSnake(position Position, direction Direction) *Snake {
	// log.Println("Creating a new snake at", position, "with direction", direction)

	snake := &Snake{}
	snake.length = 2
	snake.body = []Position{
		Position{position.X, position.Y - 1},
		position,
	}
	snake.direction = direction
	snake.dead = false

	return snake
}

// IncrementSize will increase the size (length) of the snake
func (snake *Snake) IncrementSize() {
	// log.Println("Incrementing snake size from", snake.length, "to", snake.length+1)

	snake.length++
}

// CheckHitbox will see if the supplied position is
// within the snake's calculated "body" (including head)
func (snake *Snake) CheckHitbox(position Position) bool {
	for _, p := range snake.body {
		if p == position {
			return true
		}
	}
	return false
}

// GetHead returns the snake head position
func (snake *Snake) GetHead() Position {
	return snake.body[len(snake.body)-1]
}

// UpdateDirection will set the snake direction,
// so long as it's not the opposite direction
func (snake *Snake) UpdateDirection(direction Direction) {
	// Don't allow going in opposite directions, as that would just kill the snake
	if snake.direction.X-direction.X == 0 || snake.direction.Y-direction.Y == 0 {
		return
	}
	snake.direction = direction
}

// Move the snake one coordinate along its current direction
func (snake *Snake) Move() {
	// Prepare the current and new positions
	currentPosition := snake.GetHead()
	newPosition := Position{currentPosition.X, currentPosition.Y}

	// Move one coordinate in the current direction
	switch snake.direction {
	case Direction{-1, 0}:
		newPosition.X--
	case Direction{1, 0}:
		newPosition.X++
	case Direction{0, -1}:
		newPosition.Y--
	case Direction{0, 1}:
		newPosition.Y++
	}

	// Check if this new position would result in death
	if snake.CheckHitbox(newPosition) {
		snake.dead = true
		return
	}

	// TODO: What does this do _exactly_?
	if snake.length > len(snake.body) {
		snake.body = append(snake.body, newPosition)
	} else {
		snake.body = append(snake.body[1:], newPosition)
	}
}
