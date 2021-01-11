package utils

import (
	"strconv"
)

func PadLeft(str, pad string, lenght int) string {
	for {
		str = pad + str
		if len(str) > lenght {
			return str[1:]
		}
	}
}

func FloatSliceToString(inSlice []float64) (outSlice []string) {
	for _, element := range inSlice {
		outSlice = append(outSlice, strconv.FormatFloat(element, 'f', -1, 32))
	}
	return outSlice
}

func SliceSum(data []float64) float64 {
	runSum := 0.0
	for _, element := range data {
		runSum += element
	}
	return runSum
}

func ElementWiseSum(dataIn1 []float64, dataIn2 []float64) (dataOut []float64) {
	dataOut = dataIn1
	if len(dataIn1) > 0 {
		for i, element := range dataIn2 {
			dataOut[i] += element
		}
		return dataOut
	}

	return dataIn2

}

func ElementWiseDivision(dataIn1 []float64, factor float64) (dataOut []float64) {
	dataOut = dataIn1
	if len(dataIn1) > 0 {
		for i := range dataIn1 {
			dataOut[i] *= 1 / factor
		}
		return dataOut
	}
	return dataIn1

}

func MatrixSum(data [][]float64) float64 {
	runSum := 0.0
	for _, element := range data {
		runSum += SliceSum(element)
	}
	return runSum
}

func FloatMatrixToString(data [][]float64) (output [][]string) {
	for row, rowSlice := range data {
		for column, element := range rowSlice {
			if column == 0 {
				output = append(output, []string{})
			}
			output[row] = append(output[row], strconv.FormatFloat(element, 'f', -1, 32))
		}
	}
	return output
}

func FindIntInSlice(slice []int, number int) bool {
	for _, sliceNumber := range slice {
		if sliceNumber == number {
			return true
		}
	}
	return false
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func IndexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}
