.PHONY: contracts test join demo dev-api

contracts:
	go run ./tools/gencontracts

test:
	go test ./libs/...

join:
	go run ./apps/join-cli -server=demo-survival

demo: join
	go run ./apps/join-cli -server=demo-survival -launch

dev-api:
	go run ./apps/dev-api
