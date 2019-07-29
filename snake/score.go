package snake

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Dids/clobber/util"
)

// Score information for the game
type Score struct {
	score     int
	highscore int
}

// NewScore creates and returns a new Score object
func NewScore() *Score {
	score := &Score{}
	score.score = 0
	score.highscore = score.loadHighscore()
	return score
}

// GetScore returns the current score
func (score *Score) GetScore() int {
	return score.score
}

// GetHighscore returns the best score
func (score *Score) GetHighscore() int {
	return score.highscore
}

// SetScore updates the current and best score
func (score *Score) SetScore(newScore int) {
	score.score = newScore
	if score.score > score.highscore {
		score.highscore = score.score
		score.saveHighscore(score.highscore)
	}
}

// IncrementScore updates the current and best score
func (score *Score) IncrementScore() {
	newScore := score.score + 1
	score.SetScore(newScore)
}

func (score *Score) createHighscoreFile() {
	if _, err := os.Stat(util.GetScorePath()); err != nil {
		// Create the file
		newScoreFile, err := os.Create(util.GetScorePath())
		if err != nil {
			log.Fatal("Failed to create highscore file:", err)
		}

		// Make sure to close the file when done
		defer newScoreFile.Close()

		// Write a zero score to the file
		if _, err := fmt.Fprintf(newScoreFile, "%d", 0); err != nil {
			log.Fatal("Failed to write to highscore file:", err)
		}
	}
}

func (score *Score) loadHighscore() int {
	// Make sure that the score file exists
	score.createHighscoreFile()

	// Get the current highscore from the file
	bytes, err := ioutil.ReadFile(util.GetScorePath())
	if err != nil {
		log.Fatal("Failed to read highscore file:", err)
	}

	// Parse and load the highscore
	highscoreString := string(bytes)
	highscoreString = strings.TrimSpace(highscoreString)
	if len(highscoreString) > 0 {
		highscore, err := strconv.Atoi(highscoreString)
		if err != nil {
			log.Fatal("Failed to parse highscore:", err)
		}
		return highscore
	}

	return 0
}

func (score *Score) saveHighscore(highscore int) {
	// Make sure that the score file exists
	score.createHighscoreFile()

	// Open the file
	newScoreFile, err := os.OpenFile(util.GetScorePath(), os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Fatal("Failed to open highscore file:", err)
	}

	// Make sure to close the file when done
	defer newScoreFile.Close()

	// Write the score to the file
	if _, err := fmt.Fprintf(newScoreFile, "%d\n", highscore); err != nil {
		log.Fatal("Failed to write to highscore file:", err)
	}
}
