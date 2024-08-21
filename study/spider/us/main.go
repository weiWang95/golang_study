package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.shoplazza.site/common/shoplazza-common/xid"
)

var states = []string{"AK", "AL", "AR", "AZ", "CA", "CO", "CT", "DC", "DE", "FL", "GA", "HI", "IA", "ID", "IL", "IN", "KS", "KY", "LA", "MA", "MD", "ME", "MI", "MN", "MO", "MS", "MT", "NC", "ND", "NE", "NH", "NJ", "NM", "NV", "NY", "OH", "OK", "OR", "PA", "RI", "SC", "SD", "TN", "TX", "UT", "VA", "VT", "WA", "WI", "WV", "WY"}

type Item struct {
	State   string `json:"state"`
	Country string `json:"country"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
}

func main() {
	// if err := FetchAllAreaAndZip(); err != nil {
	// 	panic(err)
	// }

	if err := FormatData(); err != nil {
		panic(err)
	}
	if err := GroupData(); err != nil {
		panic(err)
	}
}

func GroupData() error {
	f, err := os.Open("us_with_code.csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	var buf bytes.Buffer

	batchSize := 100

	var prevCityCode string
	for {
		data, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.WithStack(err)
		}

		provinceCode, city, cityCode, area, areaCode, zip := data[0], data[1], data[2], data[3], data[4], data[5]

		if batchSize == 100 {
			buf.WriteString(fmt.Sprintf("INSERT INTO `customization_area`(`id`,`store_id`,`parent_code`,`area_code`,`area_name`,`zip`) VALUES\n"))
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf("(%d,'0','%s','%s','%s','%s'),\n", xid.Get(), provinceCode, cityCode, city, ""))
				prevCityCode = cityCode
			}
			buf.WriteString(fmt.Sprintf("(%d,'0','%s','%s','%s','%s')\n", xid.Get(), cityCode, areaCode, area, zip))
		} else {
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf(",(%d,'0','%s','%s','%s','%s')\n", xid.Get(), provinceCode, cityCode, city, ""))
				prevCityCode = cityCode
			}
			buf.WriteString(fmt.Sprintf(",(%d,'0','%s','%s','%s','%s')\n", xid.Get(), cityCode, areaCode, area, zip))
		}

		batchSize--
		if batchSize == 0 {
			buf.WriteString(";\n")
			batchSize = 100
		}
	}

	w, err := os.Create("us.sql")
	if err != nil {
		return errors.WithStack(err)
	}
	defer w.Close()
	w.Write(buf.Bytes())

	return nil
}

func FormatData() error {
	f, err := os.Open("us.csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	if err := WriteCsvFile("us_with_code.csv", func(w *csv.Writer) error {
		w.Write([]string{"province_code", "city", "city_code", "area", "area_code", "zip"})

		i, j := 0, 0
		prevProvinceCode, prevCity, prevArea := "", "", ""
		for {
			data, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}

				return errors.WithStack(err)
			}

			provinceCode, city, area, zip := fmt.Sprintf("US-%s", data[0]), data[1], data[2], data[3]

			if prevProvinceCode != provinceCode {
				i, j = 0, 0
			} else if prevCity != city {
				i++
				j = 0
			} else if prevArea != area {
				j++
			}

			cityCode := fmt.Sprintf("%s-%d", provinceCode, i)
			areaCode := fmt.Sprintf("%s-%d-%d", provinceCode, i, j)

			w.Write([]string{provinceCode, city, cityCode, area, areaCode, zip})

			prevProvinceCode, prevCity, prevArea = provinceCode, city, area
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func FetchAllAreaAndZip() error {
	ch, ech := fetchAllAreaAndZip(context.TODO())

	go func() {
		fails := make([]string, 0)
		for url := range ech {
			fails = append(fails, url)
		}
		b, _ := json.Marshal(fails)

		f, err := os.Create("fail_urls.json")
		if err != nil {
			return
		}
		defer f.Close()

		f.Write(b)
	}()

	if err := WriteCsvFile("us.csv", func(w *csv.Writer) error {
		w.Write([]string{"state", "country", "city", "zip"})

		for data := range ch {
			w.Write(data)
		}
		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func fetchAllAreaAndZip(ctx context.Context) (chan []string, chan string) {
	ch := make(chan []string)
	ech := make(chan string)

	go func() {
		defer func() {
			close(ch)
			close(ech)
		}()

		q, _ := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: 10000})

		for _, state := range states {
			q.AddURL(fmt.Sprintf("https://%s.postcodebase.com/zipcode5-list", strings.ToLower(state)))
		}

		c := colly.NewCollector(
		// colly.AllowURLRevisit(),
		)
		c.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: 1,
			RandomDelay: 5 * time.Second,
		})
		// c.WithTransport(&http.Transport{
		// 	DisableKeepAlives: true,
		// })
		extensions.RandomUserAgent(c)

		// rp, err := proxy.RoundRobinProxySwitcher(ips...)
		// if err != nil {
		// 	return errors.WithStack(err)
		// }
		// c.SetProxyFunc(rp)
		// c.SetProxy("")

		c.OnRequest(func(r *colly.Request) {
			logrus.Infof("Visiting %s", r.URL.String())
		})

		c.OnError(func(r *colly.Response, err error) {
			fmt.Printf("Error: Code:%d Err:%v\n", r.StatusCode, err)
			ech <- r.Request.URL.String()
			// r.Request.Retry()

			// nr, err := r.Request.New("GET", r.Request.URL.String(), nil)
			// if err != nil {
			// 	fmt.Printf("err -> %v\n", err)
			// } else {
			// 	q.AddRequest(nr)
			// }
		})

		c.OnHTML("#block-system-main .Area table.zipcode tbody tr", func(h *colly.HTMLElement) {
			data := h.ChildTexts("td")
			if len(data) == 0 {
				return
			}

			ch <- data
		})

		c.OnHTML("#block-system-main .Area ul.pager li.pager-next", func(h *colly.HTMLElement) {
			link := h.ChildAttr("a[href]", "href")

			if link != "" {
				c.Visit(h.Request.AbsoluteURL(link))
			}
		})

		q.Run(c)
	}()

	return ch, ech
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
