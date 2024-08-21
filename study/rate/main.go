package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

func main() {
	if err := Read(); err != nil {
		panic(err)
	}
}

func Read() error {
	f, err := os.Open("data.csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	var res float64

	for {
		data, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.WithStack(err)
		}

		amount, rate := cast.ToFloat64(data[1]), cast.ToFloat64(data[2])
		res += amount * rate
	}

	fmt.Printf("%.06f\n", res)

	return nil
}
