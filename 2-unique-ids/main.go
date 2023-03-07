package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	gonanoid "github.com/matoous/go-nanoid"
)

type Request struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
}

type Response struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	MsgID int    `json:"msg_id"`
}

func main() {
	n := maelstrom.NewNode()

	n.Handle("generate", func(msg maelstrom.Message) error {
		var req Request
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		id, err := generateUniqueID()
		if err != nil {
			return err
		}

		resp := Response{
			Type:  "generate_ok",
			ID:    id,
			MsgID: req.MsgID,
		}

		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}

func generateUniqueID() (string, error) {
	// With a size of 16 and this alphabet, we have 456 years @ 100k IDs
	// generated per hour until we have a 1% probability of at least 1
	// collision, per https://zelark.github.io/nano-id-cc/.
	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
	size := 16
	return gonanoid.Generate(alphabet, size)
}
