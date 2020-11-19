package rpc

import (
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
)

type BrokerState struct {
	world [][]bool
}

type BrokerReq struct {
	initialState [][]bool
	params       gol.Params
}

type BrokerRes struct {
}

func (bs *BrokerState) broker(req BrokerReq, res *BrokerRes) (err error) {
	bs.world = req.initialState

	turn := 0
	for turn < req.params.Turns {
		// quot := req.params.ImageHeight / req.params.Threads
		// rem := req.params.ImageHeight % req.params.Threads

		// TODO: Interact via interactor
		// workers = slice of all workers
		wg := sync.WaitGroup{}
		for i := 0; i < req.params.Threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// rpc call to worker[i % numWorkers]
			}()
		}

		wg.Wait()
		// world = nextworld
		turn++

		// send event turn complete
	}
	// send event totally complete

	// stop timer?
	return nil
}
