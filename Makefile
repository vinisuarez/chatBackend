build:
	protoc -I pb/ pb/models.proto --go_out=pb