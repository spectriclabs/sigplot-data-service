package image

import (
	"math"
)

type ColorPoint struct {
	Position uint8
	Red      uint8
	Green    uint8
	Blue     uint8
}

func MakeColorPalette(controlColors []ColorPoint, numColors int) []ColorPoint {
	colorsPerPosition := numColors / 100.0
	lastPoint := controlColors[0]

	// If first control color is not at 0 then copy color for range
	lastIndexFilled := 0
	outColors := make([]ColorPoint, numColors)
	for i := 1; i < len(controlColors); i++ {
		// possDiff:=controlColors[i].position-lastPoint.position
		redDiff := (float64(controlColors[i].Red) - float64(lastPoint.Red)) * 255.0 / 100
		greenDiff := (float64(controlColors[i].Green) - float64(lastPoint.Green)) * 255.0 / 100
		blueDiff := (float64(controlColors[i].Blue) - float64(lastPoint.Blue)) * 255.0 / 100
		startRange := lastIndexFilled + 1
		endRange := int(math.Round(float64(controlColors[i].Position) * float64(colorsPerPosition)))
		for j := startRange; j < endRange; j++ {
			percentRange := (float64(j) - float64(startRange)) / float64((endRange - startRange))
			outColors[j].Red = uint8(math.Round(percentRange*redDiff + float64(lastPoint.Red)*255.0/100))
			outColors[j].Green = uint8(math.Round(percentRange*greenDiff + float64(lastPoint.Green)*255.0/100))
			outColors[j].Blue = uint8(math.Round(percentRange*blueDiff + float64(lastPoint.Blue)*255.0/100))
			outColors[j].Position = uint8(j)
			lastIndexFilled = j
		}
		lastPoint = controlColors[i]

	}
	return outColors
}

func GetColorControlPoints(colorMap string) []ColorPoint {
	switch colorMap {
	case "Greyscale":
		return []ColorPoint{
			ColorPoint{0, 0, 0, 0},
			ColorPoint{60, 50, 50, 50},
			ColorPoint{100, 100, 100, 100},
		} 
	case "RampColormap":
		return []ColorPoint{
			ColorPoint{0, 0, 0, 15},
			ColorPoint{10, 0, 0, 50},
			ColorPoint{31, 0, 65, 75},
			ColorPoint{50, 0, 80, 0},
			ColorPoint{70, 75, 80, 0},
			ColorPoint{83, 100, 60, 0},
			ColorPoint{100, 100, 0, 0},
		}
	case "ColorWheel":
		return []ColorPoint{
			ColorPoint{0, 100, 100, 0},
			ColorPoint{20, 0, 80, 40},
			ColorPoint{30, 0, 100, 100},
			ColorPoint{50, 10, 10, 0},
			ColorPoint{65, 100, 0, 0},
			ColorPoint{88, 100, 40, 0},
			ColorPoint{100, 100, 100, 0},
		}
	case "Spectrum":
		return []ColorPoint{
			ColorPoint{0, 0, 75, 0},
			ColorPoint{22, 0, 90, 90},
			ColorPoint{37, 0, 0, 85},
			ColorPoint{49, 90, 0, 85},
			ColorPoint{68, 90, 0, 0},
			ColorPoint{80, 90, 90, 0},
			ColorPoint{100, 95, 95, 95},
		}
	default:
		panic("Undefined ColorMap")
	}
}
