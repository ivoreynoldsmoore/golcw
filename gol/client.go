package gol

import (
	"bytes"
	"encoding/gob"
	"net/rpc"
)

// ClientState holds all information the client needs
type ClientState struct {
	Events chan<- Event
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
		var event Event
		buf.Write(e)
		err := dec.Decode(&event)
		HandleError(err)
		buf.Reset()
		go func() {
			cs.Events <- event
		}()
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
