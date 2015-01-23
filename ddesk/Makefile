default : check 

check:
	go build
	go vet
	go test -coverprofile=cover.out -covermode=count

