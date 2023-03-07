all: echo

echo:
	go build -o bin/echo ./echo/main.go

test-echo: echo
	./maelstrom/maelstrom test -w echo --bin ./bin/echo --node-count 1 --time-limit 10

clean:
	rm bin/*

.PHONY: all echo clean
