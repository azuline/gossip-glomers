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
	BroadcastPeerMsgType    MessageOutType = "broadcast_peer"
	BroadcastBatchMsgType   MessageOutType = "broadcast_batch"
	ReadMsgType             MessageInType  = "read"
	TopologyMsgType         MessageInType  = "topology"
	BroadcastOKMsgType      MessageOutType = "broadcast_ok"
	BroadcastPeerOKMsgType  MessageOutType = "broadcast_peer_ok"
	BroadcastBatchOKMsgType MessageOutType = "broadcast_batch_ok"
	ReadOKMsgType           MessageOutType = "read_ok"
	TopologyOKMsgType       MessageOutType = "topology_ok"
)

type Topology = map[string][]string

type BroadcastIn struct {
	MsgID   int `json:"msg_id"`
	Message int `json:"message"`
}
type BroadcastOut struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

type BroadcastPeerIn struct {
	Message int `json:"message"`
}
type BroadcastPeerToPeer struct {
	Type    MessageInType `json:"type"`
	Message int           `json:"message"`
}

type BroadcastBatchIn struct {
	Messages []int `json:"messages"`
}
type BroadcastToPeer struct {
	Type     MessageInType `json:"type"`
	Messages []int         `json:"messages"`
}

type ReadIn struct {
	MsgID int `json:"msg_id"`
}
type ReadOut struct {
	Type     MessageOutType `json:"type"`
	MsgID    int            `json:"msg_id"`
	Messages []int          `json:"messages"`
}

type TopologyIn struct {
	MsgID    int      `json:"msg_id"`
	Topology Topology `json:"topology"`
}
type TopologyOut struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

func main() {
	ctx := context.Background()
	n := maelstrom.NewNode()

	// Define read/write functions to messages. Isolate the mutex + value in a
	// private scope so that downstream code can't directly work with it.
	//
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

	// ASSUMPTION: There will be no concurrent writes to topology.
	var topology Topology

	// Kick off an asynchronous loop that syncs all messages with neighbors every second.
	go func(ctx context.Context) {
		tick := time.NewTicker(time.Second)
		for {
			select {
			case <-tick.C:
				for _, neighbor := range topology[n.ID()] {
					go func(neighbor string) {
						syncReq := BroadcastToPeer{Type: BroadcastBatchMsgType, Messages: messages()}
						if err := n.Send(neighbor, syncReq); err != nil {
							log.Fatal(err)
						}
					}(neighbor)
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	// Broadcast Peer receives a broadcast from a peer node that does not
	// require acknowledgement. Failures are acceptable for this operation.
	n.Handle(BroadcastPeerMsgType, func(msg maelstrom.Message) error {
		var req BroadcastPeerIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		appendMessage(req.Message)
		return nil
	})

	// Broadcast Batch receives a broadcast of multiple messages from a peer
	// node that does not require acknowledgement.
	n.Handle(BroadcastBatchMsgType, func(msg maelstrom.Message) error {
		var req BroadcastBatchIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		appendMessages(req.Messages)
		return nil
	})

	n.Handle(BroadcastMsgType, func(msg maelstrom.Message) error {
		var req BroadcastIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		if appendMessage(req.Message) {
			// Sends the new message to every node in the network. This is
			// intended to be called upon receiving a new message. Send a
			// message with the peer message type, which does not look for an
			// OK. If the message fails (e.g. due to network partition), it
			// will be re-sent later with the per-second sync request.
			for _, node := range n.NodeIDs() {
				syncReq := BroadcastPeerToPeer{Type: BroadcastPeerMsgType, Message: req.Message}
				if err := n.Send(node, syncReq); err != nil {
					log.Fatal(err)
				}
			}
		}
		resp := BroadcastOut{Type: BroadcastOKMsgType, MsgID: req.MsgID}
		return n.Reply(msg, resp)
	})

	n.Handle(ReadMsgType, func(msg maelstrom.Message) error {
		var req ReadIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		resp := ReadOut{Type: ReadOKMsgType, Messages: messages(), MsgID: req.MsgID}
		return n.Reply(msg, resp)
	})

	n.Handle(TopologyMsgType, func(msg maelstrom.Message) error {
		var req TopologyIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		topology = req.Topology
		resp := TopologyOut{Type: TopologyOKMsgType, MsgID: req.MsgID}
		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
