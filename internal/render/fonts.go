package render

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var jetbrainsMonoData []byte

//go:embed fonts/NotoSansMonoCJKsc-Regular.otf
var notoCJKData []byte

//go:embed fonts/Symbola.ttf
var symbolaData []byte

// Parsed font objects (immutable after init, safe for concurrent reads).
var (
	fontOnce     sync.Once
	fontJetBrain *opentype.Font
	fontNotoCJK  *opentype.Font
	fontSymbola  *opentype.Font
	fontParseErr error
)

// parseFontsLazy parses all embedded fonts once.
func parseFontsLazy() error {
	fontOnce.Do(func() {
		fontJetBrain, fontParseErr = opentype.Parse(jetbrainsMonoData)
		if fontParseErr != nil {
			return
		}
		fontNotoCJK, fontParseErr = opentype.Parse(notoCJKData)
		if fontParseErr != nil {
			return
		}
		fontSymbola, fontParseErr = opentype.Parse(symbolaData)
	})
	return fontParseErr
}

// newFaces creates font.Face objects for the given font size.
// Each call returns fresh faces — font.Face is NOT goroutine-safe.
func newFaces(fontSize float64) ([3]font.Face, error) {
	if err := parseFontsLazy(); err != nil {
		return [3]font.Face{}, err
	}

	opts := &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	}

	faceJB, err := opentype.NewFace(fontJetBrain, opts)
	if err != nil {
		return [3]font.Face{}, err
	}

	faceNoto, err := opentype.NewFace(fontNotoCJK, opts)
	if err != nil {
		return [3]font.Face{}, err
	}

	faceSym, err := opentype.NewFace(fontSymbola, opts)
	if err != nil {
		return [3]font.Face{}, err
	}

	return [3]font.Face{faceJB, faceNoto, faceSym}, nil
}

// Codepoint sets for explicit tier overrides (matching CCBot).
var notoCPs = map[rune]bool{
	0x23BF: true, // ⎿ DENTISTRY SYMBOL LIGHT VERTICAL AND BOTTOM RIGHT
}

var symbolaCPs = map[rune]bool{
	0x23F5: true, // ⏵ BLACK MEDIUM RIGHT-POINTING TRIANGLE
	0x2714: true, // ✔ HEAVY CHECK MARK
	0x274C: true, // ❌ CROSS MARK
}

// fontTier returns 0 (JetBrains), 1 (Noto CJK), or 2 (Symbola) for a rune.
// Port of CCBot's _font_tier().
func fontTier(ch rune) int {
	if symbolaCPs[ch] {
		return 2
	}
	cp := uint32(ch)
	if notoCPs[ch] ||
		(cp >= 0x2E80 &&
			(cp <= 0x9FFF ||
				(0xF900 <= cp && cp <= 0xFAFF) ||
				(0xFE30 <= cp && cp <= 0xFE4F) ||
				(0xFF00 <= cp && cp <= 0xFFEF) ||
				(0x20000 <= cp && cp <= 0x2FA1F))) {
		return 1
	}
	return 0
}

// fontSegment is a run of text that uses the same font tier.
type fontSegment struct {
	Text string
	Tier int
}

// splitByFontTier splits text into segments of consecutive characters
// sharing the same font tier. Port of CCBot's _split_line_segments_plain().
func splitByFontTier(text string) []fontSegment {
	if text == "" {
		return []fontSegment{{Text: "", Tier: 0}}
	}

	runes := []rune(text)
	var segments []fontSegment
	curTier := fontTier(runes[0])
	start := 0

	for i := 1; i < len(runes); i++ {
		tier := fontTier(runes[i])
		if tier != curTier {
			segments = append(segments, fontSegment{
				Text: string(runes[start:i]),
				Tier: curTier,
			})
			curTier = tier
			start = i
		}
	}
	segments = append(segments, fontSegment{
		Text: string(runes[start:]),
		Tier: curTier,
	})
	return segments
}
