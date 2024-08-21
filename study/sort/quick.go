package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func main() {
	// logrus.SetLevel(logrus.DebugLevel)

	d := []int{4, 1, 6, 3, 7, 2, 5, 9}
	Quick(d, 0, len(d)-1)
	fmt.Println("=> ", d)

	// SortQuick(d, 0, len(d)-1)
	// fmt.Println("=> ", d)
}

func Quick(arr []int, low, high int) {
	if low >= high {
		return
	}
	logrus.Debugf("start: %v, %d %d \n", arr[low:high+1], low, high)
	i, j := low+1, high

	for i < j {
		if arr[i] > arr[low] {
			arr[i], arr[j] = arr[j], arr[i]
			logrus.Debugf("change %v", arr[low:high+1])
			j--
		} else {
			i++
		}
	}

	if arr[i] >= arr[low] {
		i--
	}

	arr[low], arr[i] = arr[i], arr[low]

	logrus.Debugf("after: %v, %d %d, %d %d \n", arr[low:high+1], low, high, i, j)

	Quick(arr, low, i-1)
	Quick(arr, i+1, high)
}

func SortQuick(arr []int, begin, end int) {
	if begin < end {
		loc := partition(arr, begin, end)
		SortQuick(arr, begin, loc-1)
		SortQuick(arr, loc+1, end)
	}
}

func partition(arr []int, begin, end int) int {
	i, j := begin+1, end

	logrus.Debugf("start: %v, %d %d \n", arr[begin:end+1], begin, end)
	for i < j {
		if arr[i] > arr[begin] {
			arr[i], arr[j] = arr[j], arr[i]
			logrus.Debugf("change %v", arr[begin:end+1])
			j--
		} else {
			i++
		}
	}

	if arr[i] >= arr[begin] {
		i--
	}

	arr[begin], arr[i] = arr[i], arr[begin]

	logrus.Debugf("after: %v, %d %d, %d %d \n", arr[begin:end+1], begin, end, i, j)
	return i
}
