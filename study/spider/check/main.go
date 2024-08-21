package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/pkg/errors"
)

func main() {
	if err := CompareData(context.TODO()); err != nil {
		debug.PrintStack()
		panic(err)
	}
}

type Compare struct {
	C []Province `json:"C"`
	B []Province `json:"B"`
}

func CompareData(ctx context.Context) error {
	data, err := getData()
	if err != nil {
		return errors.WithStack(err)
	}

	m := make(map[string]Compare)
	for i, item := range data {
		if len(item.Provinces) == 0 {
			continue
		}

		bProvince, err := GetProvince(ctx, item.Code)
		if err != nil {
			return errors.WithStack(err)
		}

		if len(item.Provinces) != len(bProvince) {
			m[item.Code] = Compare{B: bProvince, C: data[i].Provinces}
			continue
		}

		cm := make(map[string]string, len(item.Provinces))
		for _, province := range item.Provinces {
			cm[province.Code] = province.Name
		}
		var diff bool
		for _, province := range bProvince {
			if province.Name != cm[province.Code] {
				diff = true
				break
			}
		}
		if diff {
			m[item.Code] = Compare{B: bProvince, C: data[i].Provinces}
		}
	}

	bs, err := json.Marshal(m)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := ioutil.WriteFile("compare.json", bs, os.ModePerm); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type Country struct {
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Provinces []Province `json:"provinces"`
}

type Province struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func getData() ([]Country, error) {
	bs, err := ioutil.ReadFile("data.json")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var data []Country
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	return data, nil
}

func GetProvince(ctx context.Context, country string) ([]Province, error) {
	url := fmt.Sprintf("https://winse.stg.myshoplaza.com/api/country/%s/province", country)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request error:%d", resp.StatusCode)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	var data []Province
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	return data, nil
}
