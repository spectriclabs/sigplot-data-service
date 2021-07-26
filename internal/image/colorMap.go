package image

import (
	"log"
	"math"
)

type Pixel struct {
	Position float64
	Red      float64
	Green    float64
	Blue     float64
}

func MakeColorPalette(controlColors []Pixel, numColors int) []Pixel {
	colorsPerPosition := numColors / 100.0
	lastPoint := controlColors[0]

	// If first control color is not at 0 then copy color for range
	lastIndexFilled := 0
	outColors := make([]Pixel, numColors)
	outColors[0] = Pixel{
		Red:   math.Round(controlColors[0].Red * 255.0 / 100.0),
		Blue:  math.Round(controlColors[0].Blue * 255.0 / 100.0),
		Green: math.Round(controlColors[0].Green * 255.0 / 100.0),
	}
	for controlColorIndex, controlColor := range controlColors[1:] {
		redDiff := (controlColor.Red - lastPoint.Red) * 255.0 / 100
		greenDiff := (controlColor.Green - lastPoint.Green) * 255.0 / 100
		blueDiff := (controlColor.Blue - lastPoint.Blue) * 255.0 / 100
		startRange := lastIndexFilled + 1
		endRange := int(math.Round(controlColor.Position * float64(colorsPerPosition)))
		for j := startRange; j < endRange; j++ {
			percentRange := (float64(j+1) - float64(startRange)) / float64(endRange-startRange)
			outColors[j] = Pixel{
				Position: float64(j),
				Red:      math.Round(percentRange*redDiff + float64(lastPoint.Red)*255.0/100),
				Green:    math.Round(percentRange*greenDiff + float64(lastPoint.Green)*255.0/100),
				Blue:     math.Round(percentRange*blueDiff + float64(lastPoint.Blue)*255.0/100),
			}
			lastIndexFilled = j
		}
		lastPoint = controlColors[controlColorIndex]
	}
	return outColors
}

func GetColorControlPoints(colorMap string) []Pixel {
	switch colorMap {
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
			{50, 0, 80, 0},
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
	default:
		log.Println("Unknown Colormap", colorMap, "using default RampColormap")
		return []Pixel{
			{0, 0, 0, 15},
			{10, 0, 0, 50},
			{31, 0, 65, 75},
			{50, 0, 80, 0},
			{70, 75, 80, 0},
			{83, 100, 60, 0},
			{100, 100, 0, 0},
		}
	}
}
