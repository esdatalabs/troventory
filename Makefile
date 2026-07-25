clean:
	go clean ./...

build:
	go build ./...

test:
	go test ./...

feature:
	git commit -m "feat: $(msg)"

fix: 
	git commit -m "fix: $(msg)"

breaking:
	git commit -m "feat!: $(msg)"
