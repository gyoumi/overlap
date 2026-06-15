module github.com/gyoumi/overlap

go 1.25.5

require (
	github.com/gyoumi/grove v0.0.0-20260615010139-c93324d0ab26
	warchest v0.0.0
)

require (
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

tool warchest/cmd/warchest

replace warchest => ../warchest-errors
