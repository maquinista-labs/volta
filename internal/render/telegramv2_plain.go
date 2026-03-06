package render

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// plainRenderer renders goldmark AST nodes to plain text (no formatting markers).
type plainRenderer struct{}

func newPlainRenderer() renderer.NodeRenderer {
	return &plainRenderer{}
}

func (r *plainRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Block nodes
	reg.Register(ast.KindDocument, r.noop)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)

	// Inline nodes
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.noop)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)

	// GFM extension nodes
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.noop)
	reg.Register(east.KindTableRow, r.noop)
	reg.Register(east.KindTableCell, r.noop)
	reg.Register(east.KindStrikethrough, r.noop)
	reg.Register(east.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *plainRenderer) noop(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderThematicBreak(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("———\n")
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.Write(line.Value(source))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.Write(line.Value(source))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		parent := node.Parent().(*ast.List)
		if parent.IsOrdered() {
			pos := 1
			for c := node.Parent().FirstChild(); c != node; c = c.NextSibling() {
				pos++
			}
			start := parent.Start
			if start > 0 {
				pos = start + pos - 1
			}
			w.WriteString(fmt.Sprintf("%d. ", pos))
		} else {
			w.WriteString("- ")
		}
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.Write(line.Value(source))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.Text)
		w.Write(n.Segment.Value(source))
		if n.SoftLineBreak() || n.HardLineBreak() {
			w.WriteString("\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.String)
		w.Write(n.Value)
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				w.Write(t.Segment.Value(source))
			}
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		n := node.(*ast.Link)
		w.WriteString(" (")
		w.Write(n.Destination)
		w.WriteString(")")
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		n := node.(*ast.Image)
		w.WriteString(" (")
		w.Write(n.Destination)
		w.WriteString(")")
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.AutoLink)
		w.Write(n.URL(source))
	}
	return ast.WalkContinue, nil
}

func (r *plainRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.RawHTML)
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			w.Write(seg.Value(source))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

// renderTable renders a GFM table as pipe-delimited plain text.
func (r *plainRenderer) renderTable(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	var rows [][]string
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		cells := r.collectRowCells(child, source)
		rows = append(rows, cells)
	}

	if len(rows) == 0 {
		return ast.WalkSkipChildren, nil
	}

	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	colWidths := make([]int, numCols)
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	for i, row := range rows {
		w.WriteString("| ")
		for j := 0; j < numCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			padding := colWidths[j] - len(cell)
			w.WriteString(cell)
			for p := 0; p < padding; p++ {
				w.WriteString(" ")
			}
			w.WriteString(" | ")
		}
		w.WriteString("\n")

		if i == 0 && len(rows) > 1 {
			w.WriteString("| ")
			for j := 0; j < numCols; j++ {
				for p := 0; p < colWidths[j]; p++ {
					w.WriteString("-")
				}
				w.WriteString(" | ")
			}
			w.WriteString("\n")
		}
	}

	return ast.WalkSkipChildren, nil
}

func (r *plainRenderer) collectRowCells(row ast.Node, source []byte) []string {
	var cells []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		text := r.collectPlainText(cell, source)
		cells = append(cells, strings.TrimSpace(text))
	}
	return cells
}

func (r *plainRenderer) collectPlainText(node ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Text:
			b.Write(n.Segment.Value(source))
		case *ast.String:
			b.Write(n.Value)
		case *ast.CodeSpan:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					b.Write(t.Segment.Value(source))
				}
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func (r *plainRenderer) renderTaskCheckBox(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*east.TaskCheckBox)
		if n.IsChecked {
			w.WriteString("[x] ")
		} else {
			w.WriteString("[ ] ")
		}
	}
	return ast.WalkContinue, nil
}
