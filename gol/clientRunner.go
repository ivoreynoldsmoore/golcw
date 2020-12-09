package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"time"
)

// CState is a work around for tests
// Cause rpc.Register to properly use the new state struct
var CState *ClientState

// RunClient is used an an entrypoint for tests and for the main program
func RunClient(params Params, clientPort, brokerAddr string, events chan Event, keyPresses chan rune) [][]bool {
	// Create initial connection to negotiate network parameters e.g. IPs
	tmp, err := net.Dial("tcp", brokerAddr)
	for err != nil {
		tmp, err = net.Dial("tcp", brokerAddr)
	}
	defer tmp.Close()
	time.Sleep(1 * time.Second)

	broker, err := rpc.Dial("tcp", brokerAddr)
	for err != nil {
		broker, err = rpc.Dial("tcp", brokerAddr)
	}

	lis, err := net.Listen("tcp", clientPort)
	HandleError(err)
	defer lis.Close()

	if CState == nil {
		CState = &ClientState{Events: events, Broker: broker, Params: params}
	} else {
		CState.Broker = broker
		CState.Events = events
		CState.Params = params
	}
	fmt.Println("LOG: Create new ClientState")
	rpc.Register(CState)
	go rpc.Accept(lis)

	world, c := readFile(params, events, keyPresses)
	CState.Io = c

	stopper := make(chan struct{})
	// Pass keypresses onto broker
	go func() {
		paused := false
		for {
			select {
			case <-stopper:
				return
			case e := <-keyPresses:
				var res KpBrokerRes
				broker.Call(KeypressBroker, KpBrokerReq{Event: e}, &res)
				paused = !paused
				if e == 'p' && paused {
					fmt.Printf("Paused, Turn %d\n", res.Turn)
				} else if e == 'p' {
					fmt.Println("Continuing")
				}
			}
		}
	}()

	var res1 BrokerRes
	fmt.Println("LOG: Client sending request")
	err = broker.Call(Broker, BrokerReq{InitialState: world, Params: params}, &res1)
	fmt.Println("LOG: Got final state from broker!")
	var signal struct{}
	stopper <- signal

	// Do not try and stop the broker if it returned early:
	// Likely to early return because we stopped it with k keypress
	if err == nil {
		var res2 StopBrokerRes
		err = broker.Call(StopBroker, StopBrokerReq{Restart: true}, &res2)
		HandleError(err)
		SaveWorld(res1.FinalState, params, c)
	}

	time.Sleep(1 * time.Second)
	close(events)

	// Stop SDL/tests gracefully
	fmt.Println("LOG: Client done, returning")
	return res1.FinalState
}

// SaveWorld takes a world and saves it as specified by the tests
func SaveWorld(world [][]bool, p Params, c IoChannels) {
	c.command <- ioOutput
	c.filename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] {
				c.output <- 255
			} else {
				c.output <- 0
			}
		}
	}
}
