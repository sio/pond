module secrets

go 1.20

require (
	sandbox v0.0.0
	github.com/alecthomas/kong v0.8.0
	golang.org/x/crypto v0.12.0
)

require golang.org/x/sys v0.11.0 // indirect

replace sandbox => ../sandbox
