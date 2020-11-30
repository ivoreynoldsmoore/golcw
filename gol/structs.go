package gol

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
	// Client      *rpc.Client
	// Broker      *rpc.Client
	// Workers     []*rpc.Client
}

// NetParams holds information
type NetParams struct {
	ClientAddr, ClientPort, BrokerAddr, BrokerPort string
	WorkerAddrs, WorkerPorts                       []string
}
