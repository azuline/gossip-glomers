package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type (
	MessageInType  = string
	MessageOutType = string
)

const (
	BroadcastMsgType        MessageInType  = "broadcast"
	BroadcastBatchMsgType   MessageOutType = "broadcast_batch"
	ReadMsgType             MessageInType  = "read"
	TopologyMsgType         MessageInType  = "topology"
	BroadcastOKMsgType      MessageOutType = "broadcast_ok"
	BroadcastBatchOKMsgType MessageOutType = "broadcast_batch_ok"
	ReadOKMsgType           MessageOutType = "read_ok"
	TopologyOKMsgType       MessageOutType = "topology_ok"
)

type Topology = map[string][]string

type BroadcastRequest struct {
	MsgID   int `json:"msg_id"`
	Message int `json:"message"`
}
type BroadcastResponse struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}
type BroadcastNeighborRequest struct {
	Type    MessageInType `json:"type"`
	Message int           `json:"message"`
}

type BroadcastBatchRequest struct {
	Messages []int `json:"messages"`
}
type BroadcastBatchNeighborRequest struct {
	Type     MessageInType `json:"type"`
	Messages []int         `json:"messages"`
}

type ReadRequest struct {
	MsgID int `json:"msg_id"`
}
type ReadResponse struct {
	Type     MessageOutType `json:"type"`
	MsgID    int            `json:"msg_id"`
	Messages []int          `json:"messages"`
}
type ReadNeighborRequest struct {
	Type MessageInType `json:"type"`
}

type TopologyRequest struct {
	MsgID    int      `json:"msg_id"`
	Topology Topology `json:"topology"`
}
type TopologyResponse struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

func main() {
	// TODO: Goroutine error handling :shrug:

	ctx := context.Background()
	n := maelstrom.NewNode()

	// ASSUMPTION: Topology is populated at the start of the node's lifetime.

	// Define read/write functions to internal in-memory stores. Isolate the
	// mutex + value in a private scope so that downstream code can't directly
	// work with it.

	// appendMessage returns whether or not the message was appended. We don't
	// append the message if we already had it stored.
	var appendMessage func(int) bool
	var appendMessages func([]int)
	var messages func() []int
	{
		var mu sync.RWMutex
		var stored []int
		check := make(map[int]struct{})
		appendMessage = func(value int) bool {
			mu.Lock()
			defer mu.Unlock()
			for _, m := range stored {
				if m == value {
					return false
				}
			}
			stored = append(stored, value)
			check[value] = struct{}{}
			return true
		}
		appendMessages = func(values []int) {
			mu.Lock()
			defer mu.Unlock()
			for _, v := range values {
				if _, ok := check[v]; !ok {
					stored = append(stored, v)
					check[v] = struct{}{}
				}
			}
		}
		messages = func() []int {
			mu.RLock()
			defer mu.RUnlock()
			return stored
		}
	}

	var setTopology func(Topology)
	var topology func() Topology
	{
		var mu sync.RWMutex
		var stored Topology
		setTopology = func(t Topology) {
			mu.Lock()
			defer mu.Unlock()
			stored = t
		}
		topology = func() Topology {
			mu.RLock()
			defer mu.RUnlock()
			return stored
		}
	}

	// synchronizeWithNeighbor reads a neighbor's messages and diffs their
	// messages against this node's set of messages. For the messages that this
	// node has, but the other node doesn't have, we send them over.
	synchronizeWithNeighbor := func(ctx context.Context, neighbor string) {
		readMsg, err := n.SyncRPC(ctx, neighbor, ReadNeighborRequest{Type: ReadMsgType})
		if err != nil {
			log.Fatal(err)
		}
		var readResp ReadResponse
		if err := json.Unmarshal(readMsg.Body, &readResp); err != nil {
			log.Fatal(err)
		}

		neighborHas := make(map[int]struct{})
		for _, m := range readResp.Messages {
			neighborHas[m] = struct{}{}
		}

		var messagesToBroadcast []int
		for _, m := range messages() {
			if _, ok := neighborHas[m]; !ok {
				messagesToBroadcast = append(messagesToBroadcast, m)
			}
		}

		syncReq := BroadcastBatchNeighborRequest{Type: BroadcastBatchMsgType, Messages: messagesToBroadcast}
		if err := n.Send(neighbor, syncReq); err != nil {
			log.Fatal(err)
		}
	}

	// Kick off an asynchronous loop that runs synchronizeWithNeighbor every second.
	go func(ctx context.Context) {
		tick := time.NewTicker(time.Second)
		for {
			select {
			case <-tick.C:
				for _, neighbor := range topology()[n.ID()] {
					go synchronizeWithNeighbor(ctx, neighbor)
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	n.Handle(BroadcastMsgType, func(msg maelstrom.Message) error {
		var req BroadcastRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		if appendMessage(req.Message) {
			// TODO: Potentially delete this, IDK.
			// Sends the new message to every neighboring node. This is
			// intended to be called upon receiving a new message. While this
			// message will eventually replicate across all nodes from the
			// synchronize with neighbor per-second job, we can use this
			// function to replicate with more liveness.
			go func(message int) {
				for _, neighbor := range topology()[n.ID()] {
					syncReq := BroadcastNeighborRequest{Type: BroadcastMsgType, Message: message}
					err := n.RPC(neighbor, syncReq, func(msg maelstrom.Message) error {
						return nil
					})
					if err != nil {
						log.Fatal(err)
					}
				}
			}(req.Message)
		}
		resp := BroadcastResponse{
			Type:  BroadcastOKMsgType,
			MsgID: req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	n.Handle(BroadcastBatchMsgType, func(msg maelstrom.Message) error {
		var req BroadcastBatchRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		appendMessages(req.Messages)
		return nil
	})

	n.Handle(ReadMsgType, func(msg maelstrom.Message) error {
		var req ReadRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		resp := ReadResponse{
			Type:     ReadOKMsgType,
			Messages: messages(),
			MsgID:    req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	n.Handle(TopologyMsgType, func(msg maelstrom.Message) error {
		var req TopologyRequest
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		setTopology(req.Topology)
		resp := TopologyResponse{
			Type:  TopologyOKMsgType,
			MsgID: req.MsgID,
		}
		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
