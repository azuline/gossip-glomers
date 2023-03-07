package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Request struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
	Echo  string `json:"echo"`
}

type Response struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
	Echo  string `json:"echo"`
}

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		var req Request
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		resp := Response{
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
