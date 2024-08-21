package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func main() {
	err := HandleData("KW-ar")
	fmt.Println(err)
}

func HandleData(name string) error {
	err := WriteCsvFile(fmt.Sprintf("%s-province.csv", name), func(w1 *csv.Writer) error {
		w1.Write([]string{"province_code", "province_name"})

		err := WriteCsvFile(fmt.Sprintf("%s-area.csv", name), func(w2 *csv.Writer) error {
			w2.Write([]string{"area_code", "area_name"})

			provinceMap := make(map[string]interface{})
			err := eachCsv(fmt.Sprintf("%s.csv", name), func(data []string) error {
				provinceCode, provinceName, areaCode, areaName := data[4], data[3], data[7], data[6]

				if _, ok := provinceMap[provinceCode]; !ok {
					w1.Write([]string{provinceCode, provinceName})
					provinceMap[provinceCode] = nil
				}

				w2.Write([]string{areaCode, areaName})

				return nil
			})

			return errors.WithStack(err)
		})
		return errors.WithStack(err)
	})
	return errors.WithStack(err)
}

func WriteCsvFile(filename string, fn func(w *csv.Writer) error) error {
	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "create file fail")
	}
	defer f.Close()
	f.WriteString("\xEF\xBB\xBF") // 写入一个UTF-8 BOM
	w := csv.NewWriter(f)
	defer w.Flush()

	return fn(w)
}

func ParseStringForCsv(s string) string {
	return strings.ReplaceAll(s, "\"", "\"\"")
}

func eachCsv(filename string, fn func(data []string) error) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()
	for {
		data, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.WithStack(err)
		}

		if err := fn(data); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
