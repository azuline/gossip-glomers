echo:
	go build -o bin/echo ./echo/main.go
test-echo: echo
	./maelstrom/maelstrom test -w echo --bin ./bin/echo --node-count 1 --time-limit 10

unique-ids:
	go build -o bin/unique-ids ./unique-ids/main.go
test-unique-ids: unique-ids
	./maelstrom/maelstrom test -w unique-ids --bin ./bin/unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

clean:
	rm bin/*

.PHONY: clean echo test-echo unique-ids test-unique-ids
