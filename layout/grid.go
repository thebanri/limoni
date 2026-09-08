package layout

import (
	"github.com/thebanri/limoni/core/cell"
)

type GridConstraintType uint8

const (
	gridFixed GridConstraintType = iota
	gridPercentage
	gridFraction
	gridAuto
)

type GridConstraint struct {
	Type  GridConstraintType
	Value uint16
}

// GridFixed, sabit hücre boyutunda bir kısıtlama döndürür.
func GridFixed(val uint16) GridConstraint {
	return GridConstraint{Type: gridFixed, Value: val}
}

// GridPercentage, yüzdeye dayalı bir kısıtlama döndürür.
func GridPercentage(val uint16) GridConstraint {
	return GridConstraint{Type: gridPercentage, Value: val}
}

// GridFraction, kalan alanı esnek oranlarda paylaştıran bir kısıtlama döndürür (1fr, 2fr vb.).
func GridFraction(val uint16) GridConstraint {
	return GridConstraint{Type: gridFraction, Value: val}
}

// GridAuto, kalan veya varsayılan hücre boyutunu atar.
func GridAuto() GridConstraint {
	return GridConstraint{Type: gridAuto, Value: 0}
}

type GridLayout struct {
	Columns []GridConstraint
	Rows    []GridConstraint
	Gap     uint16
}

func NewGridLayout(cols []GridConstraint, rows []GridConstraint, gap uint16) *GridLayout {
	return &GridLayout{
		Columns: cols,
		Rows:    rows,
		Gap:     gap,
	}
}

// GridArea, grid üzerindeki belirli bir hücre alanını ve onun span genişlemesini temsil eder.
type GridArea struct {
	Area   cell.Rect
	RowIdx int
	ColIdx int
	rowH   []uint16
	colW   []uint16
	rowY   []uint16
	colX   []uint16
	gap    uint16
}

// Span, mevcut hücreyi belirtilen satır ve sütun miktarı kadar genişletir (RowSpan, ColSpan).
func (ga GridArea) Span(rowSpan, colSpan int) cell.Rect {
	if rowSpan <= 0 {
		rowSpan = 1
	}
	if colSpan <= 0 {
		colSpan = 1
	}

	x := ga.Area.X
	y := ga.Area.Y

	w := uint16(0)
	actualCols := 0
	for i := 0; i < colSpan; i++ {
		cIdx := ga.ColIdx + i
		if cIdx < len(ga.colW) {
			w += ga.colW[cIdx]
			actualCols++
		}
	}
	if actualCols > 1 {
		w += uint16(actualCols-1) * ga.gap
	}

	h := uint16(0)
	actualRows := 0
	for i := 0; i < rowSpan; i++ {
		rIdx := ga.RowIdx + i
		if rIdx < len(ga.rowH) {
			h += ga.rowH[rIdx]
			actualRows++
		}
	}
	if actualRows > 1 {
		h += uint16(actualRows-1) * ga.gap
	}

	return cell.NewRect(x, y, w, h)
}

// GridAreas, GridLayout.Split çağrısının sonucunda oluşan tüm hücre alanlarını saklar.
type GridAreas struct {
	areas [][]cell.Rect
	rowH  []uint16
	colW  []uint16
	rowY  []uint16
	colX  []uint16
	gap   uint16
}

func (ga *GridAreas) Cell(row, col int) GridArea {
	if ga == nil || len(ga.areas) == 0 || row < 0 || row >= len(ga.areas) || col < 0 || col >= len(ga.areas[row]) {
		return GridArea{}
	}
	return GridArea{
		Area:   ga.areas[row][col],
		RowIdx: row,
		ColIdx: col,
		rowH:   ga.rowH,
		colW:   ga.colW,
		rowY:   ga.rowY,
		colX:   ga.colX,
		gap:    ga.gap,
	}
}

func (g *GridLayout) Split(area cell.Rect) *GridAreas {
	if len(g.Columns) == 0 || len(g.Rows) == 0 || area.Width == 0 || area.Height == 0 {
		return &GridAreas{}
	}

	// 1. Sütun Genişliklerini Çöz
	colW := solveGridConstraints(g.Columns, area.Width, g.Gap)
	// 2. Satır Yüksekliklerini Çöz
	rowH := solveGridConstraints(g.Rows, area.Height, g.Gap)

	// Sütun ve satır X/Y başlangıç konumlarını hesapla
	colX := make([]uint16, len(colW))
	currX := area.X
	for i, w := range colW {
		colX[i] = currX
		currX += w
		if i < len(colW)-1 {
			currX += g.Gap
		}
	}

	rowY := make([]uint16, len(rowH))
	currY := area.Y
	for i, h := range rowH {
		rowY[i] = currY
		currY += h
		if i < len(rowH)-1 {
			currY += g.Gap
		}
	}

	// 2D matris alanları oluştur
	areas := make([][]cell.Rect, len(rowH))
	for r := 0; r < len(rowH); r++ {
		areas[r] = make([]cell.Rect, len(colW))
		for c := 0; c < len(colW); c++ {
			areas[r][c] = cell.NewRect(colX[c], rowY[r], colW[c], rowH[r])
		}
	}

	return &GridAreas{
		areas: areas,
		rowH:  rowH,
		colW:  colW,
		rowY:  rowY,
		colX:  colX,
		gap:   g.Gap,
	}
}

func solveGridConstraints(constraints []GridConstraint, totalVal uint16, gap uint16) []uint16 {
	n := len(constraints)
	if n == 0 {
		return nil
	}

	// Toplam boşluğu (gap) güvenli şekilde çıkar
	totalGap := uint32(0)
	if n > 1 {
		totalGap = uint32(n-1) * uint32(gap)
	}
	var availableVal uint16
	if uint32(totalVal) > totalGap {
		availableVal = totalVal - uint16(totalGap)
	}

	solved := make([]uint16, n)
	remainingVal := availableVal
	totalFr := uint32(0)

	// 1. Aşama: Sabit (Fixed) ve Yüzdesel (Percentage) olanları hesapla
	for i, c := range constraints {
		switch c.Type {
		case gridFixed:
			val := c.Value
			if val > remainingVal {
				val = remainingVal
			}
			solved[i] = val
			remainingVal -= val
		case gridPercentage:
			val := uint16((uint32(availableVal) * uint32(c.Value)) / 100)
			if val > remainingVal {
				val = remainingVal
			}
			solved[i] = val
			remainingVal -= val
		case gridFraction:
			totalFr += uint32(c.Value)
		}
	}

	// 2. Aşama: Fraction (fr) ve Auto olanları esnek şekilde paylaştır
	if totalFr > 0 && remainingVal > 0 {
		var distributed uint16
		for i, c := range constraints {
			if c.Type == gridFraction {
				val := uint16((uint32(remainingVal) * uint32(c.Value)) / totalFr)
				solved[i] = val
				distributed += val
			}
		}
		// Yuvarlama farkını dağıtarak kalan tüm alanı tam doldur
		diff := remainingVal - distributed
		for i := 0; i < len(constraints) && diff > 0; i++ {
			if constraints[i].Type == gridFraction {
				solved[i]++
				diff--
			}
		}
	} else if remainingVal > 0 {
		// Eşit şekilde paylaştır (auto veya kalan alanlar)
		autoCount := uint32(0)
		for _, c := range constraints {
			if c.Type == gridAuto {
				autoCount++
			}
		}
		if autoCount > 0 {
			autoVal := uint16(uint32(remainingVal) / autoCount)
			var distributed uint16
			for i, c := range constraints {
				if c.Type == gridAuto {
					solved[i] = autoVal
					distributed += autoVal
				}
			}
			diff := remainingVal - distributed
			for i := 0; i < len(constraints) && diff > 0; i++ {
				if constraints[i].Type == gridAuto {
					solved[i]++
					diff--
				}
			}
		}
	}

	return solved
}
