package main

import (
	"math"
	"log"
)

type colorPoint struct {
	position float64
	red      float64
	green    float64
	blue     float64
}

func makeColorPalette(controlColors []colorPoint, numColors int) []colorPoint {

	colorsPerPosition := numColors / 100.0
	lastPoint := controlColors[0]
	//If first control color is not at 0 then copy color for range
	lastIndexFilled := 0
	outColors := make([]colorPoint, numColors)
	for i := 1; i < len(controlColors); i++ {
		//possDiff:=controlColors[i].position-lastPoint.position
		redDiff := (float64(controlColors[i].red) - float64(lastPoint.red)) * 255.0 / 100
		greenDiff := (float64(controlColors[i].green) - float64(lastPoint.green)) * 255.0 / 100
		blueDiff := (float64(controlColors[i].blue) - float64(lastPoint.blue)) * 255.0 / 100
		startRange := lastIndexFilled + 1
		endRange := int(math.Round(float64(controlColors[i].position) * float64(colorsPerPosition)))
		for j := startRange; j < endRange; j++ {
			percentRange := (float64(j) - float64(startRange)) / float64((endRange - startRange-1))
			outColors[j].red = (math.Round(percentRange*redDiff + float64(lastPoint.red)*255.0/100))
			outColors[j].green = (math.Round(percentRange*greenDiff + float64(lastPoint.green)*255.0/100))
			outColors[j].blue = (math.Round(percentRange*blueDiff + float64(lastPoint.blue)*255.0/100))
			outColors[j].position = float64(j)
			lastIndexFilled = j
		}
		lastPoint = controlColors[i]

	}
	return outColors
}

func getColorConrolPoints(colorMap string) []colorPoint {
	var outColors []colorPoint
	switch colorMap {
	case "Greyscale":
		outColors = make([]colorPoint, 3)
		outColors[0] = colorPoint{0, 0, 0, 0}
		outColors[1] = colorPoint{60, 50, 50, 50}
		outColors[2] = colorPoint{100, 100, 100, 100}
	case "RampColormap":
		outColors = make([]colorPoint, 7)
		outColors[0] = colorPoint{0, 0, 0, 15}
		outColors[1] = colorPoint{10, 0, 0, 50}
		outColors[2] = colorPoint{31, 0, 65, 75}
		outColors[3] = colorPoint{50, 0, 80, 0}
		outColors[4] = colorPoint{70, 75, 80, 0}
		outColors[5] = colorPoint{83, 100, 60, 0}
		outColors[6] = colorPoint{100, 100, 0, 0}
	case "ColorWheel":
		outColors = make([]colorPoint, 7)
		outColors[0] = colorPoint{0, 100, 100, 0}
		outColors[1] = colorPoint{20, 0, 80, 40}
		outColors[2] = colorPoint{30, 0, 100, 100}
		outColors[3] = colorPoint{50, 10, 10, 0}
		outColors[4] = colorPoint{65, 100, 0, 0}
		outColors[5] = colorPoint{88, 100, 40, 0}
		outColors[6] = colorPoint{100, 100, 100, 0}
	case "Spectrum":
		outColors = make([]colorPoint, 7)
		outColors[0] = colorPoint{0, 0, 75, 0}
		outColors[1] = colorPoint{22, 0, 90, 90}
		outColors[2] = colorPoint{37, 0, 0, 85}
		outColors[3] = colorPoint{49, 90, 0, 85}
		outColors[4] = colorPoint{68, 90, 0, 0}
		outColors[5] = colorPoint{80, 90, 90, 0}
		outColors[6] = colorPoint{100, 95, 95, 95}
	case "calewhite":
		outColors = make([]colorPoint, 7)
		outColors[0] =colorPoint{ 0,100,100,100}
		outColors[1] =colorPoint{ 16.666,0,0,100}
		outColors[2] =colorPoint{ 33.333,0,100,100}
		outColors[3] =colorPoint{ 50,0,100,0}
		outColors[4] =colorPoint{ 66.666,100,100,0}
		outColors[5] =colorPoint{ 83.333,100,0,0}
		outColors[6] =colorPoint{ 100,100,0,100}
	case "HotDesat":
		outColors = make([]colorPoint, 8)
		outColors[0] =colorPoint{ 0,27.84,27.84,85.88}
		outColors[1] =colorPoint{ 14.2857,0,0,35.69}
		outColors[2] =colorPoint{ 28.571,0,100,100}
		outColors[3] =colorPoint{ 42.857,0,49.8,0}
		outColors[4] =colorPoint{ 57.14286,100,100,0}
		outColors[5] =colorPoint{ 71.42857,100,37.65,0}
		outColors[6] =colorPoint{ 85.7143,41.96,0,0}
		outColors[7] =colorPoint{ 100,87.84,29.8,29.8}
	case "Sunset":
		outColors = make([]colorPoint, 7)
		outColors[0] =colorPoint{ 0,10,0,23}
		outColors[1] =colorPoint{ 18,34,0,60}
		outColors[2] =colorPoint{ 36,58,20,47}
		outColors[3] =colorPoint{ 55,74,20,28}
		outColors[4] =colorPoint{ 72,90,43,0}
		outColors[5] =colorPoint{ 87,100,72,0}
		outColors[6] =colorPoint{ 100,100,100,76}
	default:
		log.Println("Unknown Colormap",colorMap, "using default RampColormap" )
		outColors = make([]colorPoint, 7)
		outColors[0] = colorPoint{0, 0, 0, 15}
		outColors[1] = colorPoint{10, 0, 0, 50}
		outColors[2] = colorPoint{31, 0, 65, 75}
		outColors[3] = colorPoint{50, 0, 80, 0}
		outColors[4] = colorPoint{70, 75, 80, 0}
		outColors[5] = colorPoint{83, 100, 60, 0}
		outColors[6] = colorPoint{100, 100, 0, 0}
	}
	return outColors
}
