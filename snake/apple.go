package snake

// Apple is delicious
type Apple struct {
	// Position of the apple
	position Position
}

// NewApple plops a delicious apple from the nearest apple tree
func NewApple(position Position) *Apple {
	// log.Println("Creating a new apple at", position)

	apple := &Apple{}
	apple.position = position

	return apple
}
