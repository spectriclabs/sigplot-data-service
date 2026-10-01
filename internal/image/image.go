package image

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
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
			default:
				log.Println("Unknown cxmode", cxmode, "defaulting to Re")
				outData[i/2] = datain[i]
			}
		}
		return outData
	} else {
		switch cxmode {
		case "Ma":
			outData := make([]float64, len(datain))
			for i := range datain {
				outData[i] = math.Abs(datain[i])
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
			for i := range datain {
				datain[i] = 10 * math.Log10(math.Max(datain[i], loThresh))
			}
			return datain
		case "L2":
			for i := range datain {
				datain[i] = 20 * math.Log10(math.Max(datain[i], loThresh))
			}
			return datain

		}
		return datain

	}
}

func DownSampleLineInY(datain []float64, outxsize int, transform string) []float64 {
	numLines := len(datain) / outxsize
	processSlice := make([]float64, numLines)
	outData := make([]float64, outxsize)
	for x := 0; x < outxsize; x++ {
		for y := 0; y < numLines; y++ {
			processSlice[y] = datain[y*outxsize+x]
		}
		outData[x] = Transform(processSlice, transform)
	}
	return outData
}

func DownSampleLineInX(datain []float64, outxsize int, transform string, outData []float64, outLineNum int) {
	var xelementsperoutput float64
	xelementsperoutput = float64(len(datain)) / float64(outxsize)
	if xelementsperoutput > 1 {
		for x := 0; x < outxsize; x++ {
			var startelement int
			var endelement int
			if x != (outxsize - 1) {
				startelement = int(math.Round(float64(x) * xelementsperoutput))
				endelement = int(math.Round(float64(x+1) * xelementsperoutput))
			} else {
				endelement = len(datain)
				startelement = endelement - int(math.Ceil(xelementsperoutput))
			}

			outData[outLineNum*outxsize+x] = Transform(datain[startelement:endelement], transform)

		}
	} else {

		for x := 0; x < outxsize; x++ {
			index := int(math.Floor(float64(x) * xelementsperoutput))
			outData[outLineNum*outxsize+x] = datain[index]
		}
	}
}

func Transform(dataIn []float64, transform string) float64 {
	switch transform {
	case "mean":
		var sum float64
		for _, v := range dataIn {
			sum += v
		}
		num := sum / float64(len(dataIn))
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "max":
		start := 0
		num := dataIn[0]
		if num != num {
			var ok bool
			start, num, ok = firstNonNaN(dataIn[1:])
			start++
			if !ok {
				log.Println("DoTransform produced NaN")
				return 0
			}
		}
		for _, v := range dataIn[start+1:] {
			if v > num {
				num = v
			}
		}
		return num
	case "min":
		start := 0
		num := dataIn[0]
		if num != num {
			var ok bool
			start, num, ok = firstNonNaN(dataIn[1:])
			start++
			if !ok {
				log.Println("DoTransform produced NaN")
				return 0
			}
		}
		for _, v := range dataIn[start+1:] {
			if v < num {
				num = v
			}
		}
		return num
	case "maxabs":
		start := 0
		num := dataIn[0]
		if num != num {
			var ok bool
			start, num, ok = firstNonNaN(dataIn[1:])
			start++
			if !ok {
				log.Println("DoTransform produced NaN")
				return 0
			}
		}
		maxVal := math.Abs(num)
		for _, v := range dataIn[start+1:] {
			if av := math.Abs(v); av > maxVal {
				maxVal = av
			}
		}
		return maxVal
	case "first":
		num := dataIn[0]
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	default:
		log.Println("Unknown transform", transform, "using first")
		num := dataIn[0]
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num

	}
}

func firstNonNaN(data []float64) (int, float64, bool) {
	for i, v := range data {
		if v == v {
			return i, v, true
		}
	}
	return 0, math.NaN(), false
}

func CreateOutput(dataIn []float64, fileFormat string, zmin, zmax float64, colorMap string) []byte {
	dataOut := new(bytes.Buffer)
	numColors := 500
	if fileFormat == "RGBA" {
		colorPalette := GetCachedPalette(colorMap, numColors)
		fscale := float64(len(colorPalette)) / math.Abs(zmax-zmin)
		maxIndex := len(colorPalette) - 1
		output := make([]byte, len(dataIn)*4)
		for i, v := range dataIn {
			n := (v - zmin) * fscale
			var ci int
			if math.IsNaN(n) || n <= 0 {
				ci = 0
			} else if n > float64(maxIndex) {
				ci = maxIndex
			} else {
				ci = int(n)
			}
			offset := i * 4
			output[offset] = byte(colorPalette[ci].Red)
			output[offset+1] = byte(colorPalette[ci].Green)
			output[offset+2] = byte(colorPalette[ci].Blue)
			output[offset+3] = 255
		}
		return output
	} else {
		if len(fileFormat) < 2 {
			log.Println("Unsupported output type")
			return dataOut.Bytes()
		}
		log.Println("Creating Output of Type ", fileFormat)
		switch string(fileFormat[1]) {
		case "B":
			output := make([]byte, len(dataIn))
			for i, v := range dataIn {
				output[i] = byte(int8(math.Round(v)))
			}
			return output
		case "I":
			output := make([]byte, len(dataIn)*2)
			for i, v := range dataIn {
				binary.LittleEndian.PutUint16(output[i*2:], uint16(int16(math.Round(v))))
			}
			return output
		case "L":
			output := make([]byte, len(dataIn)*4)
			for i, v := range dataIn {
				binary.LittleEndian.PutUint32(output[i*4:], uint32(int32(math.Round(v))))
			}
			return output
		case "F":
			output := make([]byte, len(dataIn)*4)
			for i, v := range dataIn {
				binary.LittleEndian.PutUint32(output[i*4:], math.Float32bits(float32(v)))
			}
			return output
		case "D":
			output := make([]byte, len(dataIn)*8)
			for i, v := range dataIn {
				binary.LittleEndian.PutUint64(output[i*8:], math.Float64bits(v))
			}
			return output
		case "P":
			numBytes := (len(dataIn) + 7) / 8
			output := make([]byte, numBytes)
			for i := 0; i < numBytes; i++ {
				var b uint8
				for j := 0; j < 8; j++ {
					var bit uint8
					idx := i*8 + j
					if idx < len(dataIn) && dataIn[idx] > 0 {
						bit = 1
					} else {
						bit = 0
					}
					b = (b << 1) | bit
				}
				output[i] = b
			}
			return output
		default:
			log.Println("Unsupported output type")
		}

		return dataOut.Bytes()
	}

}
