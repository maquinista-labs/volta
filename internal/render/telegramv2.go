package render

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// telegramRenderer renders goldmark AST nodes to Telegram MarkdownV2 format.
// A fresh instance is created per convertWithGoldmark call, so mutable state is safe.
type telegramRenderer struct {
	blockquoteDepth int
}

func newTelegramRenderer() renderer.NodeRenderer {
	return &telegramRenderer{}
}

// RegisterFuncs registers render functions for each AST node kind.
func (r *telegramRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Block nodes
	reg.Register(ast.KindDocument, r.renderDocument)
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
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)

	// GFM extension nodes
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.noop)
	reg.Register(east.KindTableRow, r.noop)
	reg.Register(east.KindTableCell, r.noop)
	reg.Register(east.KindStrikethrough, r.renderStrikethrough)
	reg.Register(east.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *telegramRenderer) noop(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderDocument(w util.BufWriter, _ []byte, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("*")
	} else {
		w.WriteString("*\n")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		if r.blockquoteDepth > 0 {
			w.WriteString("\n>")
		} else {
			w.WriteString("\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderThematicBreak(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("\\—\\—\\—\n")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Indented code blocks are disabled in parser, but safety fallback
	if entering {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.WriteString(escapeMarkdownV2(string(line.Value(source))))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.FencedCodeBlock)
		w.WriteString("```")
		if lang := n.Language(source); lang != nil {
			w.Write(lang)
		}
		w.WriteString("\n")

		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.WriteString(escapeCodeContent(string(line.Value(source))))
		}
		w.WriteString("```\n")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.blockquoteDepth++
		w.WriteString(">")
	} else {
		r.blockquoteDepth--
		w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		// Add newline after list only if not at end of document
		if node.NextSibling() != nil {
			w.WriteString("\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		parent := node.Parent().(*ast.List)
		if parent.IsOrdered() {
			// Count position in list
			pos := 1
			for c := node.Parent().FirstChild(); c != node; c = c.NextSibling() {
				pos++
			}
			start := parent.Start
			if start > 0 {
				pos = start + pos - 1
			}
			w.WriteString(fmt.Sprintf("%d\\. ", pos))
		} else {
			w.WriteString("\\- ")
		}
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.WriteString(escapeMarkdownV2(string(line.Value(source))))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.Text)
		text := string(n.Segment.Value(source))
		escaped := escapeMarkdownV2(text)

		if r.blockquoteDepth > 0 {
			// Inside blockquote: prefix > after newlines
			escaped = strings.ReplaceAll(escaped, "\n", "\n>")
		}

		w.WriteString(escaped)

		if n.SoftLineBreak() {
			if r.blockquoteDepth > 0 {
				w.WriteString("\n>")
			} else {
				w.WriteString("\n")
			}
		}
		if n.HardLineBreak() {
			if r.blockquoteDepth > 0 {
				w.WriteString("\n>")
			} else {
				w.WriteString("\n")
			}
		}
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.String)
		text := string(n.Value)
		escaped := escapeMarkdownV2(text)

		if r.blockquoteDepth > 0 {
			escaped = strings.ReplaceAll(escaped, "\n", "\n>")
		}

		w.WriteString(escaped)
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("`")
		// Collect raw text from children
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				raw := string(t.Segment.Value(source))
				w.WriteString(escapeCodeContent(raw))
			}
		}
		w.WriteString("`")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	if n.Level == 2 {
		w.WriteString("*")
	} else {
		w.WriteString("_")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("[")
	} else {
		n := node.(*ast.Link)
		w.WriteString("](")
		w.WriteString(escapeURL(string(n.Destination)))
		w.WriteString(")")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("[")
	} else {
		n := node.(*ast.Image)
		w.WriteString("](")
		w.WriteString(escapeURL(string(n.Destination)))
		w.WriteString(")")
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.AutoLink)
		url := string(n.URL(source))
		w.WriteString(escapeMarkdownV2(url))
	}
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.RawHTML)
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			w.WriteString(escapeMarkdownV2(string(seg.Value(source))))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

// renderTable renders a GFM table as a code block with pipe-delimited rows.
func (r *telegramRenderer) renderTable(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	// Collect all rows (header + body)
	var rows [][]string
	var alignments []east.Alignment

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case east.KindTableHeader:
			// Get alignments from header
			if header, ok := child.(*east.TableHeader); ok {
				_ = header
			}
			cells := r.collectRowCells(child, source)
			rows = append(rows, cells)

			// Collect alignments from the header cells
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*east.TableCell); ok {
					alignments = append(alignments, tc.Alignment)
				}
			}
		case east.KindTableRow:
			cells := r.collectRowCells(child, source)
			rows = append(rows, cells)
		}
	}

	if len(rows) == 0 {
		return ast.WalkSkipChildren, nil
	}

	// Calculate column widths
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

	// Render as code block
	w.WriteString("```\n")
	for i, row := range rows {
		w.WriteString("| ")
		for j := 0; j < numCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			// Pad cell
			padding := colWidths[j] - len(cell)
			w.WriteString(cell)
			for p := 0; p < padding; p++ {
				w.WriteString(" ")
			}
			w.WriteString(" | ")
		}
		w.WriteString("\n")

		// Separator after header
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
	w.WriteString("```\n")

	return ast.WalkSkipChildren, nil
}

// collectRowCells extracts cell text from a table row node.
func (r *telegramRenderer) collectRowCells(row ast.Node, source []byte) []string {
	var cells []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		text := r.collectPlainText(cell, source)
		cells = append(cells, strings.TrimSpace(text))
	}
	return cells
}

// collectPlainText collects raw text from an AST subtree (for table cells).
func (r *telegramRenderer) collectPlainText(node ast.Node, source []byte) string {
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
			// Collect code span text
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

func (r *telegramRenderer) renderStrikethrough(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	w.WriteString("~")
	return ast.WalkContinue, nil
}

func (r *telegramRenderer) renderTaskCheckBox(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*east.TaskCheckBox)
		if n.IsChecked {
			w.WriteString("\\[x\\] ")
		} else {
			w.WriteString("\\[ \\] ")
		}
	}
	return ast.WalkContinue, nil
}
