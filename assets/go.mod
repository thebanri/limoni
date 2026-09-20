// The images in this directory are for the README, not for anyone building
// against Limoni — and a directory with its own go.mod is not part of the
// module around it. So `go get github.com/thebanri/limoni` downloads none of
// them: the module was 14.30 MB before this file and 3.70 MB after it, with
// the difference being four demo GIFs.
//
// Nothing imports this module and it holds no Go code. It exists only to draw
// that line. Relative links in README.md keep working, because GitHub serves
// them from the repository rather than from the module.
module github.com/thebanri/limoni/assets

go 1.25.0
