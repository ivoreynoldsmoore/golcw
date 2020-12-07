package gol

import (
	"bytes"
	"encoding/gob"
	"net/rpc"
)

// ClientState holds all information the client needs
type ClientState struct {
	Params Params
	Events chan Event
	Broker *rpc.Client
	Io     IoChannels
}

// ClientReq holds any data sent to the client
type ClientReq struct {
	// Would represent a []Event, but we need to gob-ify it
	Events [][]byte
}

// ClientRes is the result type for the client function
type ClientRes struct {
}

// SaveClientReq is the request type for the save client function
type SaveClientReq struct {
	World [][]bool
}

// SaveClientRes is the result type for the save client function
type SaveClientRes struct {
}

// SendEvents is called from broker to send events back to the client
func (cs *ClientState) SendEvents(req ClientReq, res *ClientRes) (err error) {
	for _, e := range req.Events {
		event, err := decodeEvent(e)
		HandleError(err)

		// Big """work-around""" for gob encoding/interface type errors
		// Event decodes into value of type Event, but really has type *Event
		// This code looks ugly, but is the only way we managed to make this decoding work
		// fmt.Println("LOG: Recv event")
		// fmt.Println(event)
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

// SaveClient is called from broker to save its current world state
func (cs *ClientState) SaveClient(req SaveClientReq, res *SaveClientRes) (err error) {
	SaveWorld(req.World, cs.Params, cs.Io)
	return nil
}

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

// EncodeEvents is a plural version of EncodeEvent
func EncodeEvents(events []Event) ([][]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	out := make([][]byte, 0)

	for _, event := range events {
		err := enc.Encode(&event)
		if err != nil {
			return nil, err
		}
		out = append(out, buf.Bytes())
		buf.Reset()
	}
	return out, nil
}

func decodeEvent(e []byte) (interface{}, error) {
	var buf bytes.Buffer
	dec := gob.NewDecoder(&buf)
	var event interface{}

	buf.Write(e)
	err := dec.Decode(&event)
	if err != nil {
		return nil, err
	}
	buf.Reset()
	return event, nil
}
