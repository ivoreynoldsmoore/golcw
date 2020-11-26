package dist

import (
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
)

// BrokerState holds all information the broker needs
type BrokerState struct {
	Params gol.Params
}

// BrokerReq is the request type for the broker function
type BrokerReq struct {
	InitialState [][]bool
}

// BrokerRes is the result type for the broker function
type BrokerRes struct {
	FinalState [][]bool
}

// Broker takes care of all communication between workers
func (bs *BrokerState) Broker(req BrokerReq, res *BrokerRes) (err error) {
	numWorkers := len(bs.Params.Workers)
	world := req.InitialState
	height := bs.Params.ImageHeight
	width := bs.Params.ImageWidth
	quot := height / numWorkers
	rem := height % numWorkers

	// Cache slice heights, annoying to recalculate
	slices := make([]int, numWorkers)
	// Last value is special case, longer than others
	for i := 0; i < numWorkers-1; i++ {
		slices[i] = quot
	}
	slices[numWorkers-1] = quot + rem

	wg := sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		var res InitWorkerRes
		req := InitWorkerReq{
			req.InitialState,
			i * quot,
			slices[i],
			height,
			width}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bs.Params.Workers[i].Call(InitWorker, req, &res)
		}(i)
	}
	wg.Wait()

	turn := 0
	for turn < bs.Params.Turns {
		nextWorld := newWorld(height, width)
		// TODO: Interact via interactor
		wg := sync.WaitGroup{}
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				var res WorkerRes
				req := WorkerReq{
					RowBelow: world[(quot*i+height-1)%height],
					RowAbove: world[(quot*i+height+slices[i])%height],
					Turn:     turn}

				bs.Params.Workers[i%numWorkers].Call(Worker, req, &res)

				// We only copy the section we're interested about
				// The rest is probably bogus
				for j := quot * i; j < quot*i+slices[i]; j++ {
					nextWorld[j] = res.World[j]
				}
			}(i)
		}

		wg.Wait()
		world = nextWorld
		turn++

		// send event turn complete
	}
	// send event totally complete

	// stop timer?
	res.FinalState = world
	return nil
}
