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

	stopper := make(chan struct{})
	fmt.Println("Worker initialised")
	rpc.Register(&WorkerState{Stopper: stopper})
	go rpc.Accept(lis)
	<-stopper
}
