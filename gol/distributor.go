package gol

import (
	"fmt"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioInput    <-chan uint8
	ioOutput   chan<- uint8
}

type executorParams struct {
	c    distributorChannels
	turn int
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	world := newWorld(p)
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.ioInput != 0
			// If it's true, we've flipped it
			if world[y][x] {
				c.events <- CellFlipped{Cell: util.Cell{X: x, Y: y}, CompletedTurns: 0}
			}
		}
	}

	turn := 0
	for turn = 0; turn < p.Turns; turn++ {
		nextWorld := newWorld(p)
		execParam := executorParams{c, turn}
		executor(execParam, 0, 0, p.ImageWidth, p.ImageHeight, world, nextWorld)
		world = nextWorld

		// 1st turn completes when i = 0, etc.
		c.events <- TurnComplete{turn + 1}
	}
	c.events <- FinalTurnComplete{Alive: calculateAliveCells(p, world), CompletedTurns: turn}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func newWorld(p Params) [][]bool {
	world := make([][]bool, p.ImageHeight)
	for x := range world {
		world[x] = make([]bool, p.ImageWidth)
	}
	return world
}
