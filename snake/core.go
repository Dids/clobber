package snake

import "fmt"

// Position in X and Y coordinates
type Position struct {
	X int
	Y int
}

// Size in Width and Hiehgt
type Size struct {
	Width  int
	Height int
}

// Direction in -1/+1 X and Y coordinates
type Direction struct {
	X int
	Y int
	// Position
}

func clearScreen() {
	fmt.Printf("\033[3J") // Clear the history
	fmt.Printf("\033[H")  // Clear the screen
}

// Equals compares this object with X and Y integers
func (pos Position) Equals(x int, y int) bool {
	return pos.X == x && pos.Y == y
}

// Equals compares this object with X and Y integers
func (dir Direction) Equals(x int, y int) bool {
	return dir.X == x && dir.Y == y
}

// Zero returns true if values are set to 0,
// otherwise it returns true
func (pos Position) Zero() bool {
	return pos.X == 0 && pos.Y == 0
}

// Zero returns true if values are set to 0,
// otherwise it returns true
func (dir Direction) Zero() bool {
	return dir.X == 0 && dir.Y == 0
}
