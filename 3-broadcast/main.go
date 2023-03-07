package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type (
	MessageInType  = string
	MessageOutType = string
)

const (
	BroadcastType   MessageInType  = "broadcast"
	ReadType        MessageInType  = "read"
	TopologyType    MessageInType  = "topology"
	BroadcastOKType MessageOutType = "broadcast_ok"
	ReadOKType      MessageOutType = "read_ok"
	TopologyOKType  MessageOutType = "topology_ok"
)

type BroadcastRequest struct {
	Message int `json:"message"`
	MsgID   int `json:"msg_id"`
}
type BroadcastResponse struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

type ReadRequest struct {
	MsgID int `json:"msg_id"`
}
type ReadResponse struct {
	Type     MessageOutType `json:"type"`
	Messages []int          `json:"messages"`
	MsgID    int            `json:"msg_id"`
}

type TopologyRequest struct {
	Topology map[string][]string `json:"topology"`
	MsgID    int                 `json:"msg_id"`
}
type TopologyResponse struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

func main() {
	n := maelstrom.NewNode()

	// Define read/write functions to the store of broadcasted messages.
	// Isolate the mutex + value in a private scope so that downstream code
	// can't directly work with it.
	var writeMessages func(int)
	var readMessages func() []int
	{
		var mu sync.RWMutex
		var messages []int
		writeMessages = func(value int) {
			mu.Lock()
			defer mu.Unlock()
			messages = append(messages, value)
		}
		readMessages = func() []int {
			mu.RLock()
			defer mu.RUnlock()
			return messages
		}
	}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var req BroadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		writeMessages(req.Message)
		resp := BroadcastResponse{
			Type:  BroadcastOKType,
			MsgID: req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var req ReadRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		resp := ReadResponse{
			Type:     ReadOKType,
			Messages: readMessages(),
			MsgID:    req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var req TopologyRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		resp := TopologyResponse{
			Type:  TopologyOKType,
			MsgID: req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
