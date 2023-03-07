all: echo

echo:
	go build -o bin/echo ./echo/main.go

clean:
	rm bin/*

.PHONY: all echo clean
