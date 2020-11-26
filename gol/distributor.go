package gol

import (
	"fmt"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	keyPresses <-chan rune
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioInput    <-chan uint8
	ioOutput   chan<- uint8
}

func timer(c distributorChannels, world *[][]bool, turn *int, mut *sync.Mutex, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			mut.Lock()
			cells := CalculateAliveCells(*world)
			c.events <- AliveCellsCount{CellsCount: len(cells), CompletedTurns: *turn}
			mut.Unlock()
		case <-stop:
			return
		}

	}
}

func loadWorld(p Params, c distributorChannels) [][]bool {
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
	return world
}

func saveWorld(p Params, c distributorChannels, world [][]bool) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] {
				c.ioOutput <- 255
			} else {
				c.ioOutput <- 0
			}
		}
	}
}

// Distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	world := loadWorld(p, c)
	mut := sync.Mutex{}
	turn := 0
	// Used to signal the timer thread to stop
	stop := make(chan struct{})
	go timer(c, &world, &turn, &mut, stop)

outer:
	for turn < p.Turns {
		execParam := executorParams{c, &sync.WaitGroup{}, turn}
		nextWorld := newWorld(p)
		quot := p.ImageHeight / p.Threads
		rem := p.ImageHeight % p.Threads

		// We handle input before the loop
		select {
		case kp := <-c.keyPresses:
			if kp == 'q' {
				break outer
			} else if kp == 's' {
				saveWorld(p, c, world)
			} else if kp == 'p' {
				fmt.Println(turn)
				// Block, discarding all non-p inputs
				for <-c.keyPresses != 'p' {
				}
				fmt.Println("Continuing")
			}
		default:
		}

		for i := 0; i < p.Threads; i++ {
			execParam.wg.Add(1)
			// On safety:
			// The executors all have one read-only copy of the previous world, which must
			// be data-safe as no data is being written. They also all have a write-only
			// copy in the form of nextWorld, but each thread should write to a different
			// part of the data, so it's fine.
			//
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
	saveWorld(p, c, world)

	c.events <- FinalTurnComplete{Alive: CalculateAliveCells(world), CompletedTurns: turn}

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
