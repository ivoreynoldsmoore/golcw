package gol

import (
	"fmt"
	"net"
	"net/rpc"
)

// RunWorker initialises and runs the worker
func RunWorker(workerPort string) {
	lis, err := net.Listen("tcp", workerPort)
	HandleError(err)
	defer lis.Close()

	fmt.Println("LOG: Worker accepting requests")
	stopper := make(chan struct{})
	rpc.Register(&WorkerState{Stopper: stopper})
	go rpc.Accept(lis)
	<-stopper
}
