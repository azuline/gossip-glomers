package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type In struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
	Echo  string `json:"echo"`
}

type Out struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
	Echo  string `json:"echo"`
}

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		var req In
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		resp := Out{
			Type:  "echo_ok",
			MsgID: req.MsgID,
			Echo:  req.Echo,
		}

		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
