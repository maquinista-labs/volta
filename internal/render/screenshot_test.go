package render

import (
	"bytes"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestRenderScreenshot_Basic(t *testing.T) {
	paneText := "Hello World\nLine 2"
	data, err := RenderScreenshot(paneText)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty PNG")
	}

	// Verify it's valid PNG
	_, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Errorf("invalid PNG: %v", err)
	}
}

func TestRenderScreenshot_WithANSI(t *testing.T) {
	paneText := "\x1b[31mRed text\x1b[0m Normal text"
	data, err := RenderScreenshot(paneText)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty PNG")
	}
}

func TestRenderScreenshot_256Color(t *testing.T) {
	paneText := "\x1b[38;5;196mBright red\x1b[0m"
	data, err := RenderScreenshot(paneText)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty PNG")
	}
}

func TestRenderScreenshot_RGBColor(t *testing.T) {
	paneText := "\x1b[38;2;255;128;0mOrange text\x1b[0m"
	data, err := RenderScreenshot(paneText)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty PNG")
	}
}

func TestRenderScreenshot_Empty(t *testing.T) {
	data, err := RenderScreenshot("")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty PNG even for empty input")
	}
}

func TestParseANSILine_Plain(t *testing.T) {
	runs := parseANSILine("Hello World")
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Text != "Hello World" {
		t.Errorf("text = %q", runs[0].Text)
	}
}

func TestParseANSILine_Color(t *testing.T) {
	runs := parseANSILine("\x1b[31mRed\x1b[0m Normal")
	if len(runs) < 2 {
		t.Fatalf("expected at least 2 runs, got %d", len(runs))
	}
	// First run should be red
	if runs[0].FG != ansi16Colors[1] {
		t.Errorf("first run FG = %v, want red", runs[0].FG)
	}
	if runs[0].Text != "Red" {
		t.Errorf("first run text = %q, want 'Red'", runs[0].Text)
	}
}

func TestParseANSILine_Bold(t *testing.T) {
	runs := parseANSILine("\x1b[1;31mBold Red\x1b[0m")
	if len(runs) < 1 {
		t.Fatal("expected at least 1 run")
	}
	if !runs[0].Bold {
		t.Error("should be bold")
	}
	// Bold red should use bright red
	if runs[0].FG != ansi16Colors[9] {
		t.Errorf("bold red FG = %v, want bright red %v", runs[0].FG, ansi16Colors[9])
	}
}

func TestParseANSILine_Background(t *testing.T) {
	runs := parseANSILine("\x1b[42mGreen BG\x1b[0m")
	if runs[0].BG != ansi16Colors[2] {
		t.Errorf("BG = %v, want green", runs[0].BG)
	}
}

func TestApplySGR_Reset(t *testing.T) {
	fg, bg, bold := applySGR("0", color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, true)
	if fg != defaultFG {
		t.Errorf("FG should reset to default")
	}
	if bg != defaultBG {
		t.Errorf("BG should reset to default")
	}
	if bold {
		t.Error("bold should reset")
	}
}

func TestApplySGR_Empty(t *testing.T) {
	fg, bg, bold := applySGR("", defaultFG, defaultBG, false)
	if fg != defaultFG || bg != defaultBG || bold {
		t.Error("empty params should reset")
	}
}

func TestColor256_SystemColors(t *testing.T) {
	for i := 0; i < 16; i++ {
		got := color256(i)
		if got != ansi16Colors[i] {
			t.Errorf("color256(%d) = %v, want %v", i, got, ansi16Colors[i])
		}
	}
}

func TestColor256_Cube(t *testing.T) {
	// Color 16 should be black (0,0,0)
	got := color256(16)
	if got != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("color256(16) = %v, want black", got)
	}

	// Color 231 should be white (255,255,255)
	got = color256(231)
	if got != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("color256(231) = %v, want white", got)
	}
}

func TestColor256_Grayscale(t *testing.T) {
	// Color 232 should be near-black
	got := color256(232)
	if got.R != 8 || got.G != 8 || got.B != 8 {
		t.Errorf("color256(232) = %v, want near-black", got)
	}

	// Color 255 should be near-white
	got = color256(255)
	if got.R != 238 || got.G != 238 || got.B != 238 {
		t.Errorf("color256(255) = %v, want near-white", got)
	}
}

func TestApplySGR_ExtendedFG256(t *testing.T) {
	fg, _, _ := applySGR("38;5;196", defaultFG, defaultBG, false)
	expected := color256(196)
	if fg != expected {
		t.Errorf("FG = %v, want %v", fg, expected)
	}
}

func TestApplySGR_ExtendedFGRGB(t *testing.T) {
	fg, _, _ := applySGR("38;2;255;128;64", defaultFG, defaultBG, false)
	expected := color.RGBA{255, 128, 64, 255}
	if fg != expected {
		t.Errorf("FG = %v, want %v", fg, expected)
	}
}

func TestApplySGR_BrightColors(t *testing.T) {
	fg, _, _ := applySGR("91", defaultFG, defaultBG, false)
	if fg != ansi16Colors[9] {
		t.Errorf("bright red FG = %v, want %v", fg, ansi16Colors[9])
	}

	_, bg, _ := applySGR("102", defaultFG, defaultBG, false)
	if bg != ansi16Colors[10] {
		t.Errorf("bright green BG = %v, want %v", bg, ansi16Colors[10])
	}
}

func TestNewFaces(t *testing.T) {
	faces, err := newFaces(28)
	if err != nil {
		t.Fatalf("newFaces failed: %v", err)
	}
	for i, face := range faces {
		if face == nil {
			t.Errorf("face[%d] is nil", i)
		}
	}
}

func TestFontTier(t *testing.T) {
	tests := []struct {
		ch   rune
		want int
	}{
		{'A', 0},         // ASCII → JetBrains
		{'z', 0},         // ASCII → JetBrains
		{'0', 0},         // digit → JetBrains
		{'─', 0},         // box drawing U+2500 → JetBrains (below 0x2E80)
		{0x23BF, 1},      // ⎿ explicit Noto override
		{0x4E00, 1},      // 一 CJK ideograph
		{0x9FFF, 1},      // last CJK unified
		{0xFF01, 1},      // ！ fullwidth exclamation
		{0x2E80, 1},      // ⺀ CJK radical
		{0x23F5, 2},      // ⏵ explicit Symbola
		{0x2714, 2},      // ✔ explicit Symbola
		{0x274C, 2},      // ❌ explicit Symbola
	}
	for _, tc := range tests {
		got := fontTier(tc.ch)
		if got != tc.want {
			t.Errorf("fontTier(%q U+%04X) = %d, want %d", tc.ch, tc.ch, got, tc.want)
		}
	}
}

func TestSplitByFontTier(t *testing.T) {
	// All ASCII
	segs := splitByFontTier("Hello")
	if len(segs) != 1 || segs[0].Text != "Hello" || segs[0].Tier != 0 {
		t.Errorf("all-ASCII: got %+v", segs)
	}

	// Empty
	segs = splitByFontTier("")
	if len(segs) != 1 || segs[0].Text != "" || segs[0].Tier != 0 {
		t.Errorf("empty: got %+v", segs)
	}

	// Mixed ASCII + CJK
	segs = splitByFontTier("Hi一二")
	if len(segs) != 2 {
		t.Fatalf("mixed: expected 2 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "Hi" || segs[0].Tier != 0 {
		t.Errorf("mixed[0]: got %+v", segs[0])
	}
	if segs[1].Text != "一二" || segs[1].Tier != 1 {
		t.Errorf("mixed[1]: got %+v", segs[1])
	}

	// Symbola character surrounded by ASCII
	segs = splitByFontTier("ok✔done")
	if len(segs) != 3 {
		t.Fatalf("symbola: expected 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[1].Tier != 2 || segs[1].Text != "✔" {
		t.Errorf("symbola[1]: got %+v", segs[1])
	}
}

func TestRenderScreenshot_ImageSize(t *testing.T) {
	// 80-column, 2-line text should produce a much larger image than with basicfont
	line := strings.Repeat("X", 80)
	paneText := line + "\n" + line
	data, err := RenderScreenshot(paneText)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	// With 28px font, 80 chars should be ~1300+ pixels wide
	if bounds.Dx() < 1000 {
		t.Errorf("image width %d is too small for 80-column text at 28px font", bounds.Dx())
	}
	// 2 lines at 39px line height + padding should be > 100px
	if bounds.Dy() < 100 {
		t.Errorf("image height %d is too small", bounds.Dy())
	}
}
