package image

import (
	"encoding/json"
	"os"
	"testing"
)

// Regenerate with: npm i sigplot@3 && node -e "global.window=global;global.navigator={userAgent:String()};const m=require('sigplot/js/m'),ColorMap=require('sigplot/js/ColorMap'),names=['Ramp Colormap','Greyscale','Color Wheel','Spectrum','calewhite','HotDesat','Sunset'],out={};for(const name of names){const cm=new ColorMap(m.Mc.colormap.find(c=>c.name===name).colors,{ncolors:500,alpha:255});out[name]=cm.map.map(c=>[c.color&255,(c.color>>>8)&255,(c.color>>>16)&255]);}const cm=new ColorMap(m.Mc.colormap.find(c=>c.name==='Ramp Colormap').colors,{ncolors:500,alpha:255});cm.setRange(0,100);out._indexTests=[-10,0,1,50,99,100,110].map(value=>{const index=cm.getColorIndex(value),c=cm.getColor(value);return{value,index,color:[c.color&255,(c.color>>>8)&255,(c.color>>>16)&255]};});console.log(JSON.stringify(out));"
func TestMakeColorPaletteMatchesSigplot(t *testing.T) {
	data, err := os.ReadFile("testdata/sigplot_reference_palettes.json")
	if err != nil {
		t.Fatalf("Failed to read reference data: %v", err)
	}

	var ref map[string]json.RawMessage
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("Failed to parse reference data: %v", err)
	}

	colormaps := []string{
		"Ramp Colormap",
		"Greyscale",
		"Color Wheel",
		"Spectrum",
		"calewhite",
		"HotDesat",
		"Sunset",
	}

	for _, name := range colormaps {
		t.Run(name, func(t *testing.T) {
			rawRef, ok := ref[name]
			if !ok {
				t.Skipf("No reference data for %s", name)
				return
			}

			var refColors [][3]int
			if err := json.Unmarshal(rawRef, &refColors); err != nil {
				t.Fatalf("Failed to parse reference colors: %v", err)
			}

			controlColors := GetColorControlPoints(name)
			palette := MakeColorPalette(controlColors, 500)

			if len(palette) != len(refColors) {
				t.Fatalf("Palette size mismatch: got %d, want %d", len(palette), len(refColors))
			}

			mismatches := 0
			for i, rc := range refColors {
				gr := int(palette[i].Red)
				gg := int(palette[i].Green)
				gb := int(palette[i].Blue)

				if gr != rc[0] || gg != rc[1] || gb != rc[2] {
					if mismatches < 10 {
						t.Errorf("Color[%d]: got rgb(%d,%d,%d), want rgb(%d,%d,%d)",
							i, gr, gg, gb, rc[0], rc[1], rc[2])
					}
					mismatches++
				}
			}
			if mismatches > 0 {
				t.Errorf("Total mismatches: %d out of %d colors", mismatches, len(refColors))
			}
		})
	}
}

func TestColorIndexMatchesSigplot(t *testing.T) {
	data, err := os.ReadFile("testdata/sigplot_reference_palettes.json")
	if err != nil {
		t.Fatalf("Failed to read reference data: %v", err)
	}

	var ref map[string]json.RawMessage
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("Failed to parse reference data: %v", err)
	}

	var indexTests []struct {
		Value int    `json:"value"`
		Index int    `json:"index"`
		Color [3]int `json:"color"`
	}
	if err := json.Unmarshal(ref["_indexTests"], &indexTests); err != nil {
		t.Fatalf("Failed to parse index tests: %v", err)
	}

	zmin := 0.0
	zmax := 100.0

	for _, tt := range indexTests {
		t.Run("", func(t *testing.T) {
			out := CreateOutput([]float64{float64(tt.Value)}, "RGBA", zmin, zmax, "Ramp Colormap")
			if len(out) != 4 {
				t.Fatalf("CreateOutput len = %d, want 4", len(out))
			}
			got := [3]int{int(out[0]), int(out[1]), int(out[2])}
			if got != tt.Color {
				t.Errorf("value=%d index=%d: got rgb(%d,%d,%d), want rgb(%d,%d,%d)",
					tt.Value, tt.Index, got[0], got[1], got[2], tt.Color[0], tt.Color[1], tt.Color[2])
			}
		})
	}
}
