package image

import (
	"log"
	"math"
	"sync"
)

type Pixel struct {
	Position float64
	Red      float64
	Green    float64
	Blue     float64
}

type paletteKey struct {
	name string
	size int
}

const maxPaletteCacheEntries = 64

var paletteCache = make(map[paletteKey][]Pixel)
var paletteMu sync.RWMutex

func GetCachedPalette(colorMap string, numColors int) []Pixel {
	canonical, known := canonicalColorMapName(colorMap)
	if !known {
		log.Println("Unknown Colormap", colorMap, "using default RampColormap")
	}
	key := paletteKey{canonical, numColors}
	paletteMu.RLock()
	if p, ok := paletteCache[key]; ok {
		paletteMu.RUnlock()
		return p
	}
	paletteMu.RUnlock()

	controlColors := GetColorControlPoints(canonical)
	palette := MakeColorPalette(controlColors, numColors)

	paletteMu.Lock()
	if _, ok := paletteCache[key]; !ok && len(paletteCache) >= maxPaletteCacheEntries {
		paletteCache = make(map[paletteKey][]Pixel)
	}
	paletteCache[key] = palette
	paletteMu.Unlock()
	return palette
}

func MakeColorPalette(controlColors []Pixel, numColors int) []Pixel {
	// Convert control colors from 0-100 percentage to 0-255 (matching sigplot's _parseColors)
	colors := make([]Pixel, len(controlColors))
	for i, c := range controlColors {
		colors[i] = Pixel{
			Position: c.Position,
			Red:      math.Floor(math.Round(255.0 * (c.Red / 100.0))),
			Green:    math.Floor(math.Round(255.0 * (c.Green / 100.0))),
			Blue:     math.Floor(math.Round(255.0 * (c.Blue / 100.0))),
		}
	}

	// Exact port of sigplot's ColorMap constructor loop
	palette := make([]Pixel, 0, numColors)
	colorindex := 1 // index into colors[] for next boundary
	colorBlockIndex := 1.0

	col1 := colors[0]
	col2 := colors[1]
	colorStop := colors[1].Position - colors[0].Position
	colorsInBlock := float64(numColors) * (colorStop / 100.0)
	factorStep := 1.0 / colorsInBlock

	for n := 0; n < numColors-2; n++ {
		if colorBlockIndex > colorsInBlock {
			col1 = colors[colorindex]
			col2Idx := colorindex + 1
			if col2Idx >= len(colors) {
				break
			}
			col2 = colors[col2Idx]
			if col1.Position >= 100 && col2.Position >= 100 {
				break
			}
			colorStop = col2.Position - col1.Position
			colorsInBlock = float64(numColors) * (colorStop / 100.0)
			factorStep = 1.0 / colorsInBlock
			colorBlockIndex = 1.0
			colorindex++
		}
		factor := factorStep * colorBlockIndex
		palette = append(palette, Pixel{
			Red:   interpolateChannel(col1.Red, col2.Red, factor),
			Green: interpolateChannel(col1.Green, col2.Green, factor),
			Blue:  interpolateChannel(col1.Blue, col2.Blue, factor),
		})
		colorBlockIndex++
	}

	// Add last control color, then prepend first (sigplot behavior)
	lastIdx := colorindex
	if lastIdx >= len(colors) {
		lastIdx = len(colors) - 1
	}
	palette = append(palette, colors[lastIdx])
	palette = append([]Pixel{colors[0]}, palette...)

	return palette
}

func interpolateChannel(a, b, factor float64) float64 {
	product := float64(factor * (b - a))
	return a + product
}

func canonicalColorMapName(colorMap string) (string, bool) {
	switch colorMap {
	case "Greyscale",
		"Ramp Colormap",
		"Color Wheel",
		"Spectrum",
		"calewhite",
		"HotDesat",
		"Sunset":
		return colorMap, true
	default:
		return "Ramp Colormap", false
	}
}

func GetColorControlPoints(colorMap string) []Pixel {
	canonical, known := canonicalColorMapName(colorMap)
	if !known {
		log.Println("Unknown Colormap", colorMap, "using default RampColormap")
	}
	switch canonical {
	case "Greyscale":
		return []Pixel{
			{0, 0, 0, 0},
			{60, 50, 50, 50},
			{100, 100, 100, 100},
		}
	case "Ramp Colormap":
		return []Pixel{
			{0, 0, 0, 15},
			{10, 0, 0, 50},
			{31, 0, 65, 75},
			{50, 0, 85, 0},
			{70, 75, 80, 0},
			{83, 100, 60, 0},
			{100, 100, 0, 0},
		}
	case "Color Wheel":
		return []Pixel{
			{0, 100, 100, 0},
			{20, 0, 80, 40},
			{30, 0, 100, 100},
			{50, 10, 10, 0},
			{65, 100, 0, 0},
			{88, 100, 40, 0},
			{100, 100, 100, 0},
		}
	case "Spectrum":
		return []Pixel{
			{0, 0, 75, 0},
			{22, 0, 90, 90},
			{37, 0, 0, 85},
			{49, 90, 0, 85},
			{68, 90, 0, 0},
			{80, 90, 90, 0},
			{100, 95, 95, 95},
		}
	case "calewhite":
		return []Pixel{
			{0, 100, 100, 100},
			{16.666, 0, 0, 100},
			{33.333, 0, 100, 100},
			{50, 0, 100, 0},
			{66.666, 100, 100, 0},
			{83.333, 100, 0, 0},
			{100, 100, 0, 100},
		}
	case "HotDesat":
		return []Pixel{
			{0, 27.84, 27.84, 85.88},
			{14.2857, 0, 0, 35.69},
			{28.571, 0, 100, 100},
			{42.857, 0, 49.8, 0},
			{57.14286, 100, 100, 0},
			{71.42857, 100, 37.65, 0},
			{85.7143, 41.96, 0, 0},
			{100, 87.84, 29.8, 29.8},
		}
	case "Sunset":
		return []Pixel{
			{0, 10, 0, 23},
			{18, 34, 0, 60},
			{36, 58, 20, 47},
			{55, 74, 20, 28},
			{72, 90, 43, 0},
			{87, 100, 72, 0},
			{100, 100, 100, 76},
		}
	}
	return nil
}
