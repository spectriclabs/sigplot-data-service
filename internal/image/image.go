package image

import (
	"bytes"
	"encoding/binary"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"log"
	"math"

	"github.com/spectriclabs/sigplot-data-service/internal/util"
)

func ApplyCXmode(datain []float64, cxmode string, complexData bool) []float64 {
	loThresh := 1.0e-20
	if complexData {
		outData := make([]float64, len(datain)/2)
		for i := 0; i < len(datain)-1; i += 2 {
			switch cxmode {
			case "Ma":
				outData[i/2] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Ph":
				outData[i/2] = math.Atan2(datain[i+1], datain[i])
			case "Re":
				outData[i/2] = datain[i]
			case "Im":
				outData[i/2] = datain[i+1]
			case "IR":
				outData[i/2] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Lo":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i/2] = 10 * math.Log10(mag2)
			case "L2":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i/2] = 20 * math.Log10(mag2)
			default: //Defaults to "real"
				log.Println("Unkown cxmode", cxmode, "defaulting to Re")
				outData[i/2] = datain[i]
			}
		}
		return outData
	} else {
		switch cxmode {
		case "Ma":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				outData[i] = math.Sqrt(datain[i] * datain[i])
			}
			return outData
		case "Ph":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				outData[i] = math.Atan2(0, datain[i])
			}
			return outData
		case "Re":
			return datain
		case "Im":
			outData := make([]float64, len(datain))
			return outData
		case "IR":
			return datain
		case "Lo":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				mag2 := math.Max(datain[i], loThresh)
				outData[i] = 10 * math.Log10(mag2)
			}
			return outData
		case "L2":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				mag2 := math.Max(datain[i], loThresh)
				outData[i] = 20 * math.Log10(mag2)
			}
			return outData

		}
		return datain //Defaults to "Real" or passthrough

	}
}

func DownSampleLineInY(datain []float64, outxsize int, transform string) []float64 {
	numLines := len(datain) / outxsize
	//log.Println("len(datain),outxsize" ,len(datain),outxsize)
	processSlice := make([]float64, numLines)
	outData := make([]float64, outxsize)
	for x := 0; x < outxsize; x++ {
		for y := 0; y < numLines; y++ {
			//log.Println("y thin" ,y,outxsize,x)
			processSlice[y] = datain[y*outxsize+x]
		}
		outData[x] = Transform(processSlice[:], transform)
	}
	return outData
}

func DownSampleLineInX(datain []float64, outxsize int, transform string, outData []float64, outLineNum int) {
	//var inputysize int =len(datain)/framesize
	var xelementsperoutput float64
	xelementsperoutput = float64(len(datain)) / float64(outxsize)
	//var thinxdata = make([]float64,outxsize)
	if xelementsperoutput > 1 { // Expansion
		for x := 0; x < outxsize; x++ {
			var startelement int
			var endelement int
			if x != (outxsize - 1) { // Not last element
				startelement = int(math.Round(float64(x) * xelementsperoutput))
				endelement = int(math.Round(float64(x+1) * xelementsperoutput))
			} else { // Last element, work backwards
				endelement = len(datain)
				startelement = endelement - int(math.Ceil(xelementsperoutput))
			}

			outData[outLineNum*outxsize+x] = Transform(datain[startelement:endelement], transform)

		}
	} else { // Expand Data by repeating input values into output

		for x := 0; x < outxsize; x++ {
			index := int(math.Floor(float64(x) * xelementsperoutput))
			outData[outLineNum*outxsize+x] = datain[index]
		}
	}
}

func Transform(dataIn []float64, transform string) float64 {
	switch transform {
	case "mean":
		num := stat.Mean(dataIn[:], nil)
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "max":
		num := floats.Max(dataIn[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "min":
		num := floats.Min(dataIn[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "maxabs":
		absnums := make([]float64, len(dataIn))
		for i := 0; i < len(dataIn); i++ {
			absnums[i] = math.Abs(dataIn[i])
		}
		num := floats.Max(absnums[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "first":
		num := dataIn[0]
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	default: // Default to first if bad value.
		log.Println("Unknown transform", transform, "using first")
		num := dataIn[0]
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num

	}
}

func CreateOutput(dataIn []float64, fileFormat string, zmin, zmax float64, colorMap string) []byte {
	// for i := 0; i < len(dataIn); i++ {
	// 	if math.IsNaN(dataIn[i]) {
	// 		log.Println("CreateOutput NaN", i)
	// 	}
	// }

	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	if fileFormat == "RGBA" {
		controlColors := GetColorControlPoints(colorMap)
		colorPalette := MakeColorPalette(controlColors, numColors)
		if zmax != zmin {
			colorsPerSpan := (zmax - zmin) / float64(numColors)
			for i := 0; i < len(dataIn); i++ {
				colorIndex := math.Round((dataIn[i]-zmin)/colorsPerSpan) - 1
				colorIndex = math.Min(math.Max(colorIndex, 0), float64(numColors-1)) //Ensure colorIndex is within the colorPalette
				a := 255
				//log.Println("colorIndex", colorIndex,dataIn[i],zmin,zmax,colorsPerSpan)
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].Red))
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].Green))
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].Blue))
				dataOut.WriteByte(byte(a))
			}
		} else {
			for i := 0; i < len(dataIn); i++ {
				a := 255
				dataOut.WriteByte(byte(colorPalette[0].Red))
				dataOut.WriteByte(byte(colorPalette[0].Green))
				dataOut.WriteByte(byte(colorPalette[0].Blue))
				dataOut.WriteByte(byte(a))
			}
		}
		//log.Println("out_data RGBA" , len(dataOut.Bytes()))
		return dataOut.Bytes()
	} else {
		log.Println("Creating Output of Type ", fileFormat)
		switch string(fileFormat[1]) {
		case "B":
			var numSlice = make([]int8, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int8(math.Round(dataIn[i]))
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			util.CheckError(err)

		case "I":
			var numSlice = make([]int16, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int16(math.Round(dataIn[i]))
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			util.CheckError(err)

		case "L":
			var numSlice = make([]int32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int32(math.Round(dataIn[i]))
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			util.CheckError(err)

		case "F":
			var numSlice = make([]float32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			util.CheckError(err)

		case "D":
			var numSlice = make([]float64, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float64(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			util.CheckError(err)

		case "P":
			extraBits := len(dataIn) % 8
			for extraBit := 0; extraBit < extraBits; extraBits++ { //Pad zeros to make the number of elements divisable by 8 so it can be packed into a byte
				dataIn = append(dataIn, 0)
			}
			numBytes := len(dataIn) / 8
			var numSlice = make([]uint8, numBytes)
			for i := 0; i < len(numSlice); i++ {
				for j := 0; j < 8; j++ {
					var bit uint8
					if dataIn[i*8+j] > 0 { //SP Data can only be 0 or 1, so if values is greater than 0, make it a 1.
						bit = 1
					} else {
						bit = 0
					}
					numSlice[i] = (numSlice[i] << 1) | bit
				}

			}
			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
			util.CheckError(err)

		default:
			log.Println("Unsupported output type")
		}
		//log.Println("out_data" , len(dataOut.Bytes()))

		return dataOut.Bytes()
	}

}
