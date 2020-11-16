package gol

import (
	"fmt"
	"sync"
	"time"

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
	wg   *sync.WaitGroup
	turn int
}

func timer(c distributorChannels, world *[][]bool, turn *int, mut *sync.Mutex, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			mut.Lock()
			cells := calculateAliveCells(*world)
			c.events <- AliveCellsCount{CellsCount: len(cells), CompletedTurns: *turn}
			mut.Unlock()
		case <-stop:
			return
		}

	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	world := newWorld(p)
	mut := sync.Mutex{}
	turn := 0

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

	stop := make(chan struct{})
	go timer(c, &world, &turn, &mut, stop)

	for turn < p.Turns {
		execParam := executorParams{c, &sync.WaitGroup{}, turn}
		nextWorld := newWorld(p)
		quot := p.ImageHeight / p.Threads
		rem := p.ImageHeight % p.Threads

		for i := 0; i < p.Threads; i++ {
			execParam.wg.Add(1)
			// On rounding issues:
			// Naively splitting up the input size via integer division doesn't work.
			// We may have a small remainder, i.e. for 512x512 with 3 threads we have...
			// ... two rows remaining. We fix this by adding those few stray rows to...
			// ... the workload of the last thread.
			if i == p.Threads-1 {
				go executor(execParam, 0, i*quot, p.ImageWidth, quot+rem, world, nextWorld)
			} else {
				go executor(execParam, 0, i*quot, p.ImageWidth, quot, world, nextWorld)
			}
		}

		execParam.wg.Wait()
		mut.Lock()
		world = nextWorld
		turn++
		mut.Unlock()

		c.events <- TurnComplete{turn}
	}
	var s struct{}
	stop <- s

	c.events <- FinalTurnComplete{Alive: calculateAliveCells(world), CompletedTurns: turn}

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
