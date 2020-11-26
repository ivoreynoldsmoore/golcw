package dist

import (
	"sync"
	"time"

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

func timer(bs *BrokerState, world *[][]bool, turn *int, mut *sync.Mutex, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			mut.Lock()
			cells := gol.CalculateAliveCells(*world)

			var err error
			var res ClientRes
			event := gol.AliveCellsCount{CellsCount: len(cells), CompletedTurns: *turn}
			events := make([][]byte, 1)
			events[0], err = EncodeEvent(event)
			if err != nil {
				panic(err)
			}
			arg := ClientReq{events}

			bs.Params.Client.Call(SendEvents, arg, &res)
			mut.Unlock()
		case <-stop:
			return
		}

	}
}

// Broker takes care of all communication between workers
func (bs *BrokerState) Broker(req BrokerReq, res *BrokerRes) (err error) {
	numWorkers := len(bs.Params.Workers)
	world := req.InitialState
	height := bs.Params.ImageHeight
	width := bs.Params.ImageWidth
	quot := height / numWorkers
	rem := height % numWorkers

	mut := sync.Mutex{}
	stop := make(chan struct{})

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
	go timer(bs, &world, &turn, &mut, stop)
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
		mut.Lock()
		world = nextWorld
		turn++
		mut.Unlock()

		// send event turn complete
	}
	// send event totally complete

	var signal struct{}
	stop <- signal
	res.FinalState = world
	return nil
}
