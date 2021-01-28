package utils

import (
	"encoding/csv"
	"os"
	"path/filepath"
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
	dataOut = make([]float64, len(dataIn2))

	if len(dataIn1) > 0 {
		copy(dataOut, dataIn1)
		for i, element := range dataIn2 {
			dataOut[i] += element
		}
		return dataOut
	} else {
		copy(dataOut, dataIn2)
	}

	return dataOut

}

func ElementWiseDivision(dataIn1 []float64, factor float64) (dataOut []float64) {
	dataOut = make([]float64, len(dataIn1))
	if len(dataIn1) > 0 {
		copy(dataOut, dataIn1)
		for i := range dataIn1 {
			dataOut[i] *= 1 / factor
		}
		return dataOut
	}
	return dataOut

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

func IndexOf(element string, data []string) (index int) {
	for k, v := range data {
		if element == v {
			index = k
			return index
		}
	}
	return -1 //not found.
}

func CheckError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}

// Exists returns whether the given file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func WriteToCSV(data [][]string, filePath string) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	CheckError(err)
	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	CheckError(err)
	defer file.Close()
}
