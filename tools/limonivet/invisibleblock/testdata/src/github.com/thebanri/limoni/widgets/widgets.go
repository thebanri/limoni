// Package widgets is a stand-in for Limoni's, with the fields the check reads.
package widgets

type Style struct{ Fg int }

type Block struct {
	Title       string
	BorderStyle Style
	Borders     uint8
	Padding     int
}

const BorderAll uint8 = 15

func NewBlock() *Block { return &Block{Borders: BorderAll} }
