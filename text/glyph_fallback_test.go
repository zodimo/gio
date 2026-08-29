package text

import (
	"testing"

	"github.com/go-text/typesetting/fontscan"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	giofont "gioui.org/font"
	"gioui.org/font/opentype"

	nsareg "eliasnaur.com/font/noto/sans/arabic/regular"
)

// arabicRune is covered by the bundled Noto Sans Arabic test font but not by
// the Go font, so shaping it with a "Go"-family query forces cross-family
// glyph fallback.
const arabicRune = '\u0628' // ب ARABIC LETTER BEH

// testFallbackCollection returns a hermetic two-font collection:
//
//   - "Go": the Go Regular face (the queried family; it has no coverage for
//     the Arabic test rune).
//   - "Noto Sans Arabic": the fallback face (it covers the test rune),
//     registered with a deliberately incorrect weight of 100.
//
// The registered weights are deliberate. For a SemiBold (600) query,
// ResolveFace's aspect-pruned fallback steps (exact family match and
// manually registered faces) keep only the closest weight. The Go face
// claims 400, so the fallback face's claimed weight of 100 loses the weight
// match and is discarded — the only fallback step that can reach it is the
// script-coverage step, which is exactly the step the fix enables by
// propagating the run's script via SetScript. Without that propagation the
// rune resolves to a face that does not cover it, producing a .notdef glyph.
func testFallbackCollection() []FontFace {
	goFace, _ := opentype.Parse(goregular.TTF)
	arFace, _ := opentype.Parse(nsareg.TTF)
	return []FontFace{
		{Font: giofont.Font{Typeface: "Go", Weight: 400}, Face: goFace},
		{Font: giofont.Font{Typeface: "Noto Sans Arabic", Weight: 100}, Face: arFace},
	}
}

// TestGlyphFallbackPropagatesScript is a regression test for cross-family
// glyph fallback.
//
// The shaper is built from a two-font collection with no system fonts, so the
// test is fully hermetic: it passes or fails solely on the fonts it registers
// itself. The "Go" family lacks the Arabic test rune, so the rune must be
// resolved by falling back to the other family.
//
// Regression: shapeText split inputs by face before computing their scripts
// and never propagated the script to the font map (SetScript). The font map's
// script-coverage fallback therefore never fired, the rune resolved to a face
// without the glyph, and it shaped as .notdef (glyph id 0) — the "tofu"
// block.
//
// The fix splits by script before splitting by face and calls SetScript per
// input in splitByFaces, making the script-coverage fallback reach the
// covering face.
func TestGlyphFallbackPropagatesScript(t *testing.T) {
	shaper := newShaperImpl(false, testFallbackCollection())

	// The query a widget like Button uses: default "Go" family at semi-bold
	// weight.
	shaper.fontMap.SetQuery(fontscan.Query{
		Families: []string{"Go"},
		Aspect:   opentype.FontToDescription(giofont.Font{Weight: giofont.SemiBold}).Aspect,
	})

	out := shaper.shapeText(fixed.I(14), english, []rune{arabicRune})
	if len(out) == 0 || len(out[0].Glyphs) == 0 {
		t.Fatalf("shapeText produced no glyphs for %U", arabicRune)
	}
	if got := out[0].Glyphs[0].GlyphID; got == 0 {
		t.Fatalf("glyph for %U is .notdef (id 0): cross-family fallback did not fire; "+
			"the run script was not propagated to the font map", arabicRune)
	}
}

// TestCollectionFaceReachableByFamily is a sanity check that the fallback
// face is registered correctly: querying its family directly must resolve the
// rune regardless of the script-propagation fix.
func TestCollectionFaceReachableByFamily(t *testing.T) {
	shaper := newShaperImpl(false, testFallbackCollection())

	shaper.fontMap.SetQuery(fontscan.Query{
		Families: []string{"Noto Sans Arabic"},
		Aspect:   opentype.FontToDescription(giofont.Font{Weight: giofont.SemiBold}).Aspect,
	})

	out := shaper.shapeText(fixed.I(14), english, []rune{arabicRune})
	if len(out) == 0 || len(out[0].Glyphs) == 0 {
		t.Fatalf("shapeText produced no glyphs for %U", arabicRune)
	}
	if got := out[0].Glyphs[0].GlyphID; got == 0 {
		t.Fatalf("glyph for %U is .notdef (id 0) despite an explicit family query", arabicRune)
	}
}