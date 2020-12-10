package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"
)

// BState is a work around for recreating RPC connections
var BState *BrokerState

// RunBroker initialises and runs the broker
func RunBroker(params Params, clientPort, brokerPort string, workerAddrs []string) {
	stopper := make(chan bool)

	workers := make([]*rpc.Client, len(workerAddrs))
	for idx, worker := range workerAddrs {
		var err error
		workers[idx], err = rpc.Dial("tcp", worker)
		for err != nil {
			workers[idx], err = rpc.Dial("tcp", worker)
		}
	}

	// Workaround for tests
	// Tests require constant reconnecting and disconnecting client-broker
	restart := true
	for restart {
		l, err := net.Listen("tcp", brokerPort)
		HandleError(err)
		tmp, err := l.Accept()
		HandleError(err)
		clientAddr := strings.Split(tmp.RemoteAddr().String(), ":")[0] + clientPort
		tmp.Close()
		l.Close()

		lis, err := net.Listen("tcp", brokerPort)
		HandleError(err)

		client, err := rpc.Dial("tcp", clientAddr)
		for err != nil {
			client, err = rpc.Dial("tcp", clientAddr)
		}

		if BState == nil {
			BState = &BrokerState{Stopper: stopper, Client: client, Workers: workers}
		} else {
			BState.Client = client
			BState.Stopper = stopper
			BState.Workers = workers

			BState.Suspend = false
			BState.Terminate = false
		}
		fmt.Println("Broker initialised")
		rpc.Register(BState)
		go rpc.Accept(lis)

		// Break and exit if not restarting, recv via Stop procedure
		restart = <-stopper
		lis.Close()
		// client.Close()
	}
}
