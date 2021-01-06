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

func SliceSum(data []float64) float64 {
	runSum := 0.0
	for _, element := range data {
		runSum += element
	}
	return runSum
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
			output[row][column] = strconv.FormatFloat(element, 'f', -1, 32)
		}
	}
	return output
}
