package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

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

	readIntDefault0 := func(node string) (int, error) {
		nodeValue, err := kv.ReadInt(ctx, node)
		if err != nil {
			var rpcErr *maelstrom.RPCError
			if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.KeyDoesNotExist {
				return 0, nil
			}
			return 0, err
		}
		return nodeValue, nil
	}

	n.Handle(AddMsgType, func(msg maelstrom.Message) error {
		var req AddIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		// Keep trying to increment the value atomically until we succeed.
		for {
			from, err := readIntDefault0(n.ID())
			if err != nil {
				return err
			}
			to := from + req.Delta
			if err := kv.CompareAndSwap(ctx, n.ID(), from, to, true); err != nil {
				var rpcErr *maelstrom.RPCError
				if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.PreconditionFailed {
					continue
				}
				return err
			}
			break
		}
		return n.Reply(msg, AddOut{Type: AddOKMsgType, MsgID: req.MsgID})
	})

	n.Handle(ReadMsgType, func(msg maelstrom.Message) error {
		var req ReadIn
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}
		ctr := 0
		for _, node := range n.NodeIDs() {
			nodeValue, err := readIntDefault0(node)
			if err != nil {
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
