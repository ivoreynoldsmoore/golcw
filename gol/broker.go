package gol

import (
	"net/rpc"
	"sync"
	"time"
)

// BrokerState holds all information the broker needs
type BrokerState struct {
	// Params  Params
	Client  *rpc.Client
	Workers []*rpc.Client
}

// BrokerReq is the request type for the broker function
type BrokerReq struct {
	InitialState [][]bool
	Params       Params
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
			cells := CalculateAliveCells(*world)
			encodeAndSendEvent(bs, AliveCellsCount{CellsCount: len(cells), CompletedTurns: *turn})
			mut.Unlock()
		case <-stop:
			return
		}
	}
}

func encodeAndSendEvent(bs *BrokerState, event Event) {
	// bytes, err := EncodeEvent(event)
	// HandleError(err)

	// req := ClientReq{Events: [][]byte{bytes}}
	// var res ClientRes
	// // err = bs.Client.Call(SendEvents, req, &res)
	// fmt.Println("LOG: Event sent")
	// HandleError(err)
}

// Broker takes care of all communication between workers
func (bs *BrokerState) Broker(req BrokerReq, res *BrokerRes) (err error) {
	numWorkers := len(bs.Workers)
	world := req.InitialState
	height := req.Params.ImageHeight
	width := req.Params.ImageWidth
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

	// Initialise all workers
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
			bs.Workers[i].Call(InitWorker, req, &res)
		}(i)
	}
	wg.Wait()

	turn := 0
	go timer(bs, &world, &turn, &mut, stop)
	// Main loop
	for turn < req.Params.Turns {
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

				err := bs.Workers[i%numWorkers].Call(Worker, req, &res)
				HandleError(err)

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
	encodeAndSendEvent(bs, FinalTurnComplete{CompletedTurns: turn, Alive: CalculateAliveCells(world)})

	var signal struct{}
	stop <- signal
	res.FinalState = world

	mut.Lock()
	mut.Unlock()
	return nil
}
