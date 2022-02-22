.PHONY: all clean

all: check_restic

check_restic: main.go
	go build

clean:
	rm -f check_restic
