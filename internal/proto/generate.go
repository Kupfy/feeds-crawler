package proto

//go:generate protoc --proto_path=./ --go_out=../../pkg/grpc/ingredientparser --go_opt=paths=source_relative --go-grpc_out=../../pkg/grpc/ingredientparser --go-grpc_opt=paths=source_relative ingredientparser.proto
