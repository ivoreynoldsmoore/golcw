package gol

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
)

// BrokerState holds all information the broker needs
type BrokerState struct {
	Stopper chan bool
	Client  *rpc.Client
	Workers []*rpc.Client
	// Internal
	// Protects suspend and cond
	Mutex   sync.Mutex
	Suspend bool
	Cond    sync.Cond
	World   [][]bool
	Turn    int
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

// StopBrokerReq is the request type for the stop broker function
type StopBrokerReq struct {
	// If we should restart the broker (tests) or just shut it down
	Restart bool
}

// StopBrokerRes is the result type for the stop broker function
type StopBrokerRes struct {
}

// KpBrokerReq is the request type for the stop broker function
type KpBrokerReq struct {
	Event rune
}

// KpBrokerRes is the result type for the stop broker function
type KpBrokerRes struct {
	Turn int
}

// StopBroker causes the broker to shut down and restart
func (bs *BrokerState) StopBroker(req StopBrokerReq, res *StopBrokerRes) (err error) {
	if !req.Restart {
		for _, worker := range bs.Workers {
			worker.Call(StopWorker, StopWorkerReq{}, &StopWorkerRes{})
		}
	}
	bs.Stopper <- req.Restart
	return nil
}

// Broker takes care of all communication between workers
func (bs *BrokerState) Broker(req BrokerReq, res *BrokerRes) (err error) {
	fmt.Println("LOG: NEW BROKER")
	numWorkers := len(bs.Workers)
	world := req.InitialState
	height := req.Params.ImageHeight
	width := req.Params.ImageWidth
	quot := height / numWorkers
	rem := height % numWorkers

	bs.Mutex = sync.Mutex{}
	bs.Cond = *sync.NewCond(&bs.Mutex)
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
	go timer(bs, stop)
	// Main loop
	for turn < req.Params.Turns {
		// Check if paused
		bs.Mutex.Lock()
		for bs.Suspend {
			bs.Cond.Wait()
		}
		bs.Mutex.Unlock()

		nextWorld := newWorld(height, width)
		flipped := make([]Event, 0)
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
				for _, flippedCell := range res.Flipped {
					flipped = append(flipped, CellFlipped{CompletedTurns: turn, Cell: flippedCell})
				}
				encodeAndSendEvents(bs, flipped)

				// We only copy the section we're interested about
				// The rest is probably bogus
				for j := quot * i; j < quot*i+slices[i]; j++ {
					nextWorld[j] = res.World[j]
				}
			}(i)
		}

		wg.Wait()
		bs.Mutex.Lock()
		world = nextWorld
		bs.World = world
		turn++
		bs.Turn = turn
		bs.Mutex.Unlock()

		encodeAndSendEvent(bs, TurnComplete{CompletedTurns: turn})
	}

	var signal struct{}
	stop <- signal
	encodeAndSendEvent(bs, FinalTurnComplete{CompletedTurns: turn, Alive: CalculateAliveCells(world)})

	res.FinalState = world

	bs.Mutex.Lock()
	bs.Mutex.Unlock()
	return nil
}

// KeypressBroker is called whenever a keypress event is sent by the controller that has to be handled
func (bs *BrokerState) KeypressBroker(req KpBrokerReq, res *KpBrokerRes) (err error) {
	bs.Mutex.Lock()
	res.Turn = bs.Turn

	switch req.Event {
	// Save world
	case 's':
		bs.Client.Call(SaveClient, SaveClientReq{World: bs.World}, &SaveClientRes{})
		bs.Mutex.Unlock()
	// New controller
	case 'q':

		bs.Mutex.Unlock()
	// Pause processing
	case 'p':
		if bs.Suspend {
			bs.Suspend = false
			bs.Mutex.Unlock()
			bs.Cond.Broadcast()
		} else {
			bs.Suspend = true
			bs.Mutex.Unlock()
		}
	// Graceful shutdown
	case 'k':
		bs.Client.Call(SaveClient, SaveClientReq{World: bs.World}, &SaveClientRes{})
		bs.StopBroker(StopBrokerReq{Restart: false}, &StopBrokerRes{})
		bs.Mutex.Unlock()
	default:
		bs.Mutex.Unlock()
	}
	return nil
}

func timer(bs *BrokerState, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			// Lock because main broker process could be updating bs.World/bs.Turn
			bs.Mutex.Lock()
			cells := CalculateAliveCells(bs.World)
			encodeAndSendEvent(bs, AliveCellsCount{CellsCount: len(cells), CompletedTurns: bs.Turn})
			bs.Mutex.Unlock()
		case <-stop:
			return
		}
	}
}

func encodeAndSendEvent(bs *BrokerState, event Event) {
	bytes, err := EncodeEvent(event)
	HandleError(err)

	req := ClientReq{Events: [][]byte{bytes}}
	var res ClientRes
	err = bs.Client.Call(SendEvents, req, &res)
	HandleError(err)
}

func encodeAndSendEvents(bs *BrokerState, events []Event) {
	eventBytes := make([][]byte, 0)
	for _, event := range events {
		eventByte, err := EncodeEvent(event)
		HandleError(err)
		eventBytes = append(eventBytes, eventByte)
	}

	req := ClientReq{Events: eventBytes}
	var res ClientRes
	err := bs.Client.Call(SendEvents, req, &res)
	HandleError(err)
}
