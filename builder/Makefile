default : check all

check:
	go build
	go vet
	go test -coverprofile=cover.out -covermode=count
