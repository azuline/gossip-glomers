package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type (
	MessageInType  = string
	MessageOutType = string
)

const (
	AddMsgType    MessageInType  = "add"
	ReadMsgType   MessageOutType = "read"
	AddOKMsgType  MessageInType  = "add_ok"
	ReadOKMsgType MessageOutType = "read_ok"
)

type AddIn struct {
	MsgID int `json:"msg_id"`
	Delta int `json:"delta"`
}
type AddOut struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
}

type ReadIn struct {
	MsgID int `json:"msg_id"`
}
type ReadOut struct {
	Type  MessageOutType `json:"type"`
	MsgID int            `json:"msg_id"`
	Value int            `json:"value"`
}

func main() {
	ctx := context.Background()

	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)

	var value func() int
	var incrValue func(int)
	{
		var mu sync.RWMutex
		var stored int
		value = func() int {
			mu.RLock()
			defer mu.RUnlock()
			return stored
		}
		incrValue = func(delta int) {
			mu.Lock()
			defer mu.Unlock()
			stored += delta
		}
	}

	n.Handle(AddMsgType, func(msg maelstrom.Message) error {
		var req AddIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		incrValue(req.Delta)
		if err := kv.Write(ctx, n.ID(), value()); err != nil {
			return err
		}
		return n.Reply(msg, AddOut{Type: AddOKMsgType, MsgID: req.MsgID})
	})

	n.Handle(ReadMsgType, func(msg maelstrom.Message) error {
		var req ReadIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		ctr := value()
		for _, node := range n.NodeIDs() {
			// We already fetched our own value from memory; skip fetch our own value from the KV store.
			if node == n.ID() {
				continue
			}
			nodeValue, err := kv.ReadInt(ctx, node)
			if err != nil {
				var rpcErr *maelstrom.RPCError
				if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.KeyDoesNotExist {
					continue
				}
				return err
			}
			ctr += nodeValue
		}
		return n.Reply(msg, ReadOut{Type: ReadOKMsgType, MsgID: req.MsgID, Value: ctr})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
