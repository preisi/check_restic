.PHONY: all clean

all: check_restic

check_restic: main.go
	CGO_ENABLED=0 go build

clean:
	rm -f check_restic
