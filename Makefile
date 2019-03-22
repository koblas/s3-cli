SRC=*.go

s3-cli: $(SRC)
	go build -o $@ $(SRC)

clean: $(SRC)
	rm -f s3-cli

test:
	go test
