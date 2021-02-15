.PHONY: proto

proto:
	@protoc \
  	--go-grpc_out=paths=source_relative:. \
    --go_out=paths=source_relative:. \
  	proto/service.proto