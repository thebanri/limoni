module github.com/thebanri/limoni

// Limoni targets the oldest Go release its dependency graph allows.
// golang.org/x/sys v0.47.0 declares go 1.25.0, so 1.25 is the honest floor:
// declaring anything lower would still fail to build for those users.
go 1.25.0

require (
	golang.org/x/crypto v0.55.0
	golang.org/x/sys v0.47.0
)
