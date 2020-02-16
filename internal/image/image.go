package image

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func CreateOutput(dataIn []float64, fileFormatString string, zmin, zmax float64, colorMap string) []byte {
	for i := 0; i < len(dataIn); i++ {
		if math.IsNaN(dataIn[i]) {
			log.Println("createOutput NaN", i)
		}
	}

	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	//var dataOut []byte
	if fileFormatString == "RGBA" {
		controlColors := GetColorControlPoints(colorMap)
		colorPalette := MakeColorPalette(controlColors, numColors)
		if zmax != zmin {
			colorsPerSpan := (zmax - zmin) / float64(numColors)
			for i := 0; i < len(dataIn); i++ {
				// Check for dataIn[i] is NaN
				colorIndex := math.Round((dataIn[i] - zmin) / colorsPerSpan)
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
		log.Println("Processing for Type ", fileFormatString)
		switch string(fileFormatString[1]) {
		case "B":
			var numSlice = make([]int8, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int8(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			Check(err)

		case "I":
			var numSlice = make([]int16, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int16(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			Check(err)

		case "L":
			var numSlice = make([]int32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			Check(err)

		case "F":
			var numSlice = make([]float32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			Check(err)

		case "D":
			var numSlice = make([]float64, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float64(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			Check(err)

		default:
			log.Println("Unsupported output type")
		}

		//TODO for SP: Add a case for P. Need to pack in 8 numbers back into 1 byte

		return dataOut.Bytes()
	}
}
