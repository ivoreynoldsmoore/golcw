package gol

import (
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
	Mutex         sync.Mutex
	Suspend       bool
	Terminate     bool
	Cond          sync.Cond
	World         [][]bool
	Turn          int
	Height, Width int
	// We set this to true when q pressed, but it is false by default
	// This will not change value whenever a new RPC "instance" is created
	// As such we will use this to check if we should accept the client's new world
	Reconnect      bool
	InitialRequest Params
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
	// Ignore new parameters and continue
	if bs.Reconnect {
		req.InitialState = bs.World
		req.Params = bs.InitialRequest
	} else {
		bs.InitialRequest = req.Params
		bs.Turn = 0
	}

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

	// Send initial cell flipped events
	flipped := make([]Event, 0)
	for _, flippedCell := range CalculateAliveCells(world) {
		flipped = append(flipped, CellFlipped{CompletedTurns: 0, Cell: flippedCell})
	}
	encodeAndSendEvents(bs, flipped)

	go timer(bs, stop)
	// Main loop
	for bs.Turn < req.Params.Turns {
		// Check if paused
		bs.Mutex.Lock()
		for bs.Suspend {
			bs.Cond.Wait()
		}
		if bs.Terminate {
			var signal struct{}
			stop <- signal
			res.FinalState = world
			return nil
		}
		bs.Mutex.Unlock()

		nextWorld := newWorld(height, width)
		// TODO: Interact via interactor
		wg := sync.WaitGroup{}
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				flipped := make([]Event, 0)
				var res WorkerRes
				req := WorkerReq{
					RowBelow: world[(quot*i+height-1)%height],
					RowAbove: world[(quot*i+height+slices[i])%height],
					Turn:     bs.Turn}

				err := bs.Workers[i%numWorkers].Call(Worker, req, &res)
				HandleError(err)
				for _, flippedCell := range res.Flipped {
					flipped = append(flipped, CellFlipped{CompletedTurns: bs.Turn, Cell: flippedCell})
				}
				go encodeAndSendEvents(bs, flipped)

				// We only copy the section we're interested about
				for j := quot * i; j < quot*i+slices[i]; j++ {
					// nextWorld[j] = res.World[j]
					nextWorld[j] = res.World[j-quot*i]
				}
			}(i)
		}

		wg.Wait()
		bs.Mutex.Lock()
		world = nextWorld
		bs.World = world
		bs.Turn++
		bs.Mutex.Unlock()

		encodeAndSendEvent(bs, TurnComplete{CompletedTurns: bs.Turn})
	}

	var signal struct{}
	stop <- signal
	encodeAndSendEvent(bs, FinalTurnComplete{CompletedTurns: bs.Turn, Alive: CalculateAliveCells(world)})

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
		bs.Reconnect = true
		bs.Terminate = true
		// bs.StopBroker(StopBrokerReq{Restart: true}, &StopBrokerRes{})
		bs.Mutex.Unlock()
	// Pause processing
	case 'p':
		if bs.Suspend {
			bs.Suspend = false
			bs.Cond.Broadcast()
			bs.Mutex.Unlock()
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
	bs.Client.Call(SendEvents, req, &res)
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
	bs.Client.Call(SendEvents, req, &res)
}
