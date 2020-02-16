package numerical

import (
	"math"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

func SuppressNaN(num float64) float64 {
	if math.IsNaN(num) {
		return 0
	}
	return num
}

func Transform(dataIn []float64, transform string) float64 {
	switch transform {
	case "mean":
		return SuppressNaN(stat.Mean(dataIn[:], nil))
	case "max":
		return SuppressNaN(floats.Max(dataIn[:]))
	case "min":
		return SuppressNaN(floats.Min(dataIn[:]))
	case "absmax":
		// num := floats.Max(math.Abs(dataIn[:]))
		return SuppressNaN(dataIn[0]) // TODO Fix
	case "first":
		return SuppressNaN(dataIn[0])
	default:
		return 0
	}
}

func DownSampleLineInX(datain []float64, outxsize int, transform string, outData []float64, outLineNum int) {
	//var inputysize int =len(datain)/framesize
	xelementsperoutput := float64(len(datain)) / float64(outxsize)
	//var thinxdata = make([]float64,outxsize)
	if xelementsperoutput > 1 {
		xElementsPerOutputCeil := int(math.Ceil(xelementsperoutput))
		//log.Println("x thin" ,xelementsperoutput,xelementsperoutput_ceil,len(datain),outxsize)

		for x := 0; x < outxsize; x++ {
			var startelement int
			var endelement int
			if x != (outxsize - 1) {
				startelement = int(math.Round(float64(x) * xelementsperoutput))
				endelement = startelement + xElementsPerOutputCeil

			} else {
				endelement = len(datain)
				startelement = endelement - xElementsPerOutputCeil
			}

			//log.Println("x thin" , x,xelementsperoutput,len(datain),outxsize,startelement,endelement)
			//out_data[x] =numerical.Transform(datain[startelement:endelement],transform)
			//log.Println("thinxdata[x]", thinxdata[x])
			outData[outLineNum*outxsize+x] = Transform(datain[startelement:endelement], transform)

		}
	} else { // Expand Data by repeating input values into output
		for x := 0; x < outxsize; x++ {
			index := int(math.Floor(float64(x) * xelementsperoutput))
			outData[outLineNum*outxsize+x] = datain[index]
		}
	}
}

func DownSampleLineInY(datain []float64, outxsize int, transform string) []float64 {
	numLines := len(datain) / outxsize
	// log.Println("len(datain),outxsize" ,len(datain),outxsize)
	processSlice := make([]float64, numLines)
	outData := make([]float64, outxsize)
	for x := 0; x < outxsize; x++ {
		for y := 0; y < numLines; y++ {
			// log.Println("y thin" ,y,outxsize,x)
			processSlice[y] = datain[y*outxsize+x]
		}
		outData[x] = Transform(processSlice[:], transform)
	}
	return outData
}

func ApplyCXmode(datain []float64, cxmode string, complexData bool) ([]float64, float64, float64) {
	loThresh := 1.0e-20
	var zmax float64 = math.Inf(-1)
	var zmin float64 = math.Inf(1)
	if complexData {
		outData := make([]float64, len(datain)/2)
		for i := 0; i < len(datain)-1; i += 2 {
			switch cxmode {
			case "Ma":
				outData[i] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Ph":
				outData[i] = math.Atan2(datain[i+1], datain[i])
			case "Re":
				outData[i] = datain[i]
			case "Im":
				outData[i] = datain[i+1]
			case "IR":
				outData[i] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Lo":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 10 * math.Log10(mag2)
			case "L2:":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 20 * math.Log10(mag2)
			}
			if outData[i] > zmax {
				zmax = outData[i]
			}
			if outData[i] < zmin {
				zmin = outData[i]
			}
		}
		return outData, zmin, zmax
	} else {
		outData := make([]float64, len(datain))
		for i := 0; i < len(datain); i++ {
			switch cxmode {
			case "Ma":
				outData[i] = math.Abs(datain[i])
			case "Ph":
				outData[i] = math.Atan2(0, datain[i])
			case "Re":
				outData[i] = datain[i]
			case "Im":
				outData[i] = 0
			case "IR":
				outData[i] = datain[i]
			case "Lo":
				mag2 := datain[i] * datain[i]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 10 * math.Log10(mag2)
			case "L2":
				mag2 := datain[i] * datain[i]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 20 * math.Log10(mag2)
			}
			if outData[i] > zmax {
				zmax = outData[i]
			}
			if outData[i] < zmin {
				zmin = outData[i]
			}
		}
		return outData, zmin, zmax
	}
}
