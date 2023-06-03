CONFIG_PATH = $(shell pwd)/certifs/
TLS_CONFIG = $(shell pwd)/tlsConfig

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert:
	cfssl gencert \
		-initca ${TLS_CONFIG}/ca-csr.json | cfssljson -bare ca
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=${TLS_CONFIG}/ca-config.json \
		-profile=client \
		${TLS_CONFIG}/client-csr.json | cfssljson -bare client
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=${TLS_CONFIG}/ca-config.json \
		-profile=server \
		${TLS_CONFIG}/server-csr.json | cfssljson -bare server
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: compile
compile: 
	protoc ./api/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
