module github.com/gyoumi/overlap

go 1.25.5

require github.com/gyoumi/grove v0.0.0

require (
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	warchest v0.0.0 // indirect
)

tool warchest/cmd/warchest

replace github.com/gyoumi/grove => ../grove

replace warchest => ../warchest
