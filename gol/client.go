package gol

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/rpc"
)

// ClientState holds all information the client needs
type ClientState struct {
	Events chan Event
	Broker *rpc.Client
}

// ClientReq holds any data sent to the client
type ClientReq struct {
	// Would represent a []Event, but we need to gob-ify it
	Events [][]byte
}

// ClientRes is the result type for the client function
type ClientRes struct {
}

// SendEvents is called from broker to send events back to the client
func (cs *ClientState) SendEvents(req ClientReq, res *ClientRes) (err error) {
	var buf bytes.Buffer
	dec := gob.NewDecoder(&buf)

	for _, e := range req.Events {
		var event interface{}
		buf.Write(e)
		err := dec.Decode(&event)
		HandleError(err)
		buf.Reset()

		// Big """work-around""" for gob encoding/interface type errors
		// Event decodes into value of type Event, but really has type *Event
		// This code looks ugly, but is the only way we managed to make this decoding work
		fmt.Println("LOG: Recv event")
		fmt.Println(event)
		switch e := event.(type) {
		case *AliveCellsCount:
			cs.Events <- *e
		case *CellFlipped:
			cs.Events <- *e
		case *ImageOutputComplete:
			cs.Events <- *e
		case *StateChange:
			cs.Events <- *e
		case *TurnComplete:
			cs.Events <- *e
		case *FinalTurnComplete:
			cs.Events <- *e
		default:
			panic("Could not decode event")
		}
	}
	return nil
}

// func encodeAndSendEvents

// EncodeEvent encodes an event in a gob encoding so that it can be send via RPC
func EncodeEvent(event Event) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&event)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
