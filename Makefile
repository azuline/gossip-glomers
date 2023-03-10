echo:
	go build -o bin/echo ./1-echo/main.go
test-echo: echo
	./maelstrom/maelstrom test -w echo --bin ./bin/echo --node-count 1 --time-limit 10

unique-ids:
	go build -o bin/unique-ids ./2-unique-ids/main.go
test-unique-ids: unique-ids
	./maelstrom/maelstrom test -w unique-ids --bin ./bin/unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

broadcast:
	go build -o bin/broadcast ./3-broadcast/main.go
test-broadcast-a: broadcast
	./maelstrom/maelstrom test -w broadcast --bin ./bin/broadcast --node-count 1 --time-limit 20 --rate 10
test-broadcast-b: broadcast
	./maelstrom/maelstrom test -w broadcast --bin ./bin/broadcast --node-count 5 --time-limit 20 --rate 10
test-broadcast-c: broadcast
	./maelstrom/maelstrom test -w broadcast --bin ./bin/broadcast --node-count 5 --time-limit 20 --rate 10 --nemesis partition
test-broadcast-d: broadcast
	./maelstrom/maelstrom test -w broadcast --bin ./bin/broadcast --node-count 25 --time-limit 20 --rate 100 --latency 100

counter:
	go build -o bin/counter ./4-counter/main.go
test-counter: counter
	./maelstrom/maelstrom test -w g-counter --bin ./bin/counter --node-count 3 --rate 100 --time-limit 20 --nemesis partition

clean:
	rm bin/*
serve:
	./maelstrom/maelstrom serve

.PHONY: clean echo test-echo unique-ids test-unique-ids
