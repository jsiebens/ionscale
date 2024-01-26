init:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@latest

generate:
	templ generate
	buf generate proto

format:
	buf format -w proto

lint:
	buf lint proto

breaking:
	buf breaking proto --against https://github.com/jsiebens/ionscale.git#subdir=proto
