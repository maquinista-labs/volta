package render

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Default colors for terminal rendering.
var (
	defaultBG = color.RGBA{30, 30, 30, 255}
	defaultFG = color.RGBA{212, 212, 212, 255}
)

// ANSI 16-color palette (standard + bright).
var ansi16Colors = [16]color.RGBA{
	{0, 0, 0, 255},       // 0 black
	{205, 49, 49, 255},   // 1 red
	{13, 188, 121, 255},  // 2 green
	{229, 229, 16, 255},  // 3 yellow
	{36, 114, 200, 255},  // 4 blue
	{188, 63, 188, 255},  // 5 magenta
	{17, 168, 205, 255},  // 6 cyan
	{229, 229, 229, 255}, // 7 white
	{102, 102, 102, 255}, // 8 bright black
	{241, 76, 76, 255},   // 9 bright red
	{35, 209, 139, 255},  // 10 bright green
	{245, 245, 67, 255},  // 11 bright yellow
	{59, 142, 234, 255},  // 12 bright blue
	{214, 112, 214, 255}, // 13 bright magenta
	{41, 184, 219, 255},  // 14 bright cyan
	{255, 255, 255, 255}, // 15 bright white
}

// styledRun is a sequence of characters with the same style.
type styledRun struct {
	Text string
	FG   color.RGBA
	BG   color.RGBA
	Bold bool
}

var reANSI = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

const (
	fontSize   = 28.0
	lineHeight = 39 // int(fontSize * 1.4), matching CCBot
	padding    = 16
)

// RenderScreenshot renders ANSI terminal text to a PNG image.
func RenderScreenshot(paneText string) ([]byte, error) {
	faces, err := newFaces(fontSize)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(paneText, "\n")

	// Parse each line into styled runs
	var parsedLines [][]styledRun
	for _, line := range lines {
		runs := parseANSILine(line)
		parsedLines = append(parsedLines, runs)
	}

	// Measure: find the widest line using the primary font's advance width.
	// Use JetBrains Mono (monospace) advance for consistent column width.
	primaryFace := faces[0]
	metrics := primaryFace.Metrics()
	ascent := metrics.Ascent.Ceil()

	// Measure char width from the primary face (monospace — all glyphs same width)
	charWidth := font.MeasureString(primaryFace, "M").Ceil()

	maxCols := 0
	for _, runs := range parsedLines {
		cols := 0
		for _, run := range runs {
			cols += len([]rune(run.Text))
		}
		if cols > maxCols {
			maxCols = cols
		}
	}

	imgWidth := maxCols*charWidth + padding*2
	imgHeight := len(parsedLines)*lineHeight + padding*2

	if imgWidth < 100 {
		imgWidth = 100
	}
	if imgHeight < 50 {
		imgHeight = 50
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Fill background using draw.Draw (faster than pixel loop for large images)
	draw.Draw(img, img.Bounds(), image.NewUniform(defaultBG), image.Point{}, draw.Src)

	// Render text
	for lineIdx, runs := range parsedLines {
		x := padding
		baseY := padding + lineIdx*lineHeight + ascent

		for _, run := range runs {
			// Split each styled run by font tier for fallback rendering
			segments := splitByFontTier(run.Text)

			for _, seg := range segments {
				face := faces[seg.Tier]

				for _, ch := range seg.Text {
					// Draw background rect if non-default
					if run.BG != defaultBG {
						bgRect := image.Rect(x, padding+lineIdx*lineHeight, x+charWidth, padding+(lineIdx+1)*lineHeight)
						draw.Draw(img, bgRect, image.NewUniform(run.BG), image.Point{}, draw.Src)
					}

					// Draw character
					d := &font.Drawer{
						Dst:  img,
						Src:  image.NewUniform(run.FG),
						Face: face,
						Dot:  fixed.P(x, baseY),
					}
					d.DrawString(string(ch))
					x += charWidth
				}
			}
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseANSILine parses a line with ANSI escape sequences into styled runs.
func parseANSILine(line string) []styledRun {
	var runs []styledRun

	fg := defaultFG
	bg := defaultBG
	bold := false

	indices := reANSI.FindAllStringSubmatchIndex(line, -1)
	lastEnd := 0

	for _, loc := range indices {
		// Text before this escape sequence
		if loc[0] > lastEnd {
			text := line[lastEnd:loc[0]]
			if text != "" {
				runs = append(runs, styledRun{Text: text, FG: fg, BG: bg, Bold: bold})
			}
		}

		// Parse the SGR parameters
		params := line[loc[2]:loc[3]]
		fg, bg, bold = applySGR(params, fg, bg, bold)
		lastEnd = loc[1]
	}

	// Remaining text
	if lastEnd < len(line) {
		text := line[lastEnd:]
		if text != "" {
			runs = append(runs, styledRun{Text: text, FG: fg, BG: bg, Bold: bold})
		}
	}

	if len(runs) == 0 {
		runs = append(runs, styledRun{Text: "", FG: fg, BG: bg})
	}

	return runs
}

// applySGR applies SGR (Select Graphic Rendition) parameters.
func applySGR(params string, fg, bg color.RGBA, bold bool) (color.RGBA, color.RGBA, bool) {
	if params == "" || params == "0" {
		return defaultFG, defaultBG, false
	}

	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

		switch {
		case n == 0: // reset
			fg = defaultFG
			bg = defaultBG
			bold = false
		case n == 1: // bold
			bold = true
		case n >= 30 && n <= 37: // standard FG
			idx := n - 30
			if bold {
				idx += 8
			}
			fg = ansi16Colors[idx]
		case n >= 40 && n <= 47: // standard BG
			bg = ansi16Colors[n-40]
		case n == 38: // extended FG
			if i+1 < len(parts) {
				mode, _ := strconv.Atoi(parts[i+1])
				if mode == 5 && i+2 < len(parts) {
					// 256-color
					colorIdx, _ := strconv.Atoi(parts[i+2])
					fg = color256(colorIdx)
					i += 2
				} else if mode == 2 && i+4 < len(parts) {
					// RGB
					r, _ := strconv.Atoi(parts[i+2])
					g, _ := strconv.Atoi(parts[i+3])
					b, _ := strconv.Atoi(parts[i+4])
					fg = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
					i += 4
				}
			}
		case n == 48: // extended BG
			if i+1 < len(parts) {
				mode, _ := strconv.Atoi(parts[i+1])
				if mode == 5 && i+2 < len(parts) {
					colorIdx, _ := strconv.Atoi(parts[i+2])
					bg = color256(colorIdx)
					i += 2
				} else if mode == 2 && i+4 < len(parts) {
					r, _ := strconv.Atoi(parts[i+2])
					g, _ := strconv.Atoi(parts[i+3])
					b, _ := strconv.Atoi(parts[i+4])
					bg = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
					i += 4
				}
			}
		case n == 39: // default FG
			fg = defaultFG
		case n == 49: // default BG
			bg = defaultBG
		case n >= 90 && n <= 97: // bright FG
			fg = ansi16Colors[n-90+8]
		case n >= 100 && n <= 107: // bright BG
			bg = ansi16Colors[n-100+8]
		}
	}

	return fg, bg, bold
}

// color256 returns a color from the 256-color palette.
func color256(idx int) color.RGBA {
	if idx < 16 {
		return ansi16Colors[idx]
	}
	if idx < 232 {
		// 6x6x6 color cube
		idx -= 16
		b := idx % 6
		idx /= 6
		g := idx % 6
		r := idx / 6
		return color.RGBA{
			uint8(r * 51),
			uint8(g * 51),
			uint8(b * 51),
			255,
		}
	}
	// Grayscale ramp (232-255)
	gray := uint8((idx-232)*10 + 8)
	return color.RGBA{gray, gray, gray, 255}
}
