package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.shoplazza.site/common/shoplazza-common/xid"
)

var countryName = "can"
var countryCode = "CA"
var plProvince = map[string]string{
	"Alberta":               "CA-AB",
	"British Columbia":      "CA-BC",
	"Manitoba":              "CA-MB",
	"New Brunswick":         "CA-NB",
	"Newfoundland":          "CA-NL",
	"Northwest Territories": "CA-NT",
	"Nova Scotia":           "CA-NS",
	"Nunavut":               "CA-NU",
	"Ontario":               "CA-ON",
	"Prince Edward Island":  "CA-PE",
	"Quebec":                "CA-QC",
	"Saskatchewan":          "CA-SK",
	"Yukon Territory":       "CA-YT",
}

type Area struct {
	Parent string `json:"parent"`
	Name   string `json:"name"`
	Link   string `json:"link"`
}

func main() {
	// if err := FetchAllCity(); err != nil {
	// 	panic(err)
	// }

	// if err := FetchAllAreaAndZip(); err != nil {
	// 	panic(err)
	// }

	// if err := FormatJsonData(); err != nil {
	// 	panic(err)
	// }

	if err := FormatData(); err != nil {
		panic(err)
	}

	if err := GroupData(); err != nil {
		panic(err)
	}

}

type Item struct {
	Name string `json:"name"`
}

func GroupData() error {
	f, err := os.Open(countryName + "_with_code.csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	var buf bytes.Buffer

	batchSize := 100

	var prevProvinceCode string
	var prevCityCode string
	for {
		data, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.WithStack(err)
		}

		province, provinceCode, city, cityCode, area, areaCode := data[0], data[1], data[2], data[3], "", ""
		if len(data) >= 5 {
			area, areaCode = data[4], data[5]
		}

		if prevProvinceCode != provinceCode {
			if buf.Len() != 0 {
				buf.WriteString(";\n")
			}
			buf.WriteString(fmt.Sprintf(`INSERT INTO province(store_id,country_code2,name,cn_name,code) VALUES('%s','%s',"%s","%s",'%s');`, "", countryCode, province, province, provinceCode))
			buf.WriteRune('\n')
			prevProvinceCode = provinceCode
			batchSize = 100
		}
		if batchSize == 100 {
			buf.WriteString("INSERT INTO `customization_area`(`id`,`store_id`,`parent_code`,`area_code`,`area_name`) VALUES\n")
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf(`(%d,'%s','%s','%s',"%s")`, xid.Get(), "0", provinceCode, cityCode, city))
				if areaCode != "" {
					buf.WriteRune(',')
				}
				buf.WriteRune('\n')
				prevCityCode = cityCode
			}
			if areaCode != "" {
				buf.WriteString(fmt.Sprintf(`(%d,'%s','%s','%s',"%s")`, xid.Get(), "0", cityCode, areaCode, area))
				buf.WriteRune('\n')
			}
		} else {
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf(`,(%d,'%s','%s','%s',"%s")`, xid.Get(), "0", provinceCode, cityCode, city))
				buf.WriteRune('\n')
				prevCityCode = cityCode
			}
			if areaCode != "" {
				buf.WriteString(fmt.Sprintf(`,(%d,'%s','%s','%s',"%s")`, xid.Get(), "0", cityCode, areaCode, area))
				buf.WriteRune('\n')
			}
		}

		batchSize--
		if batchSize == 0 {
			buf.WriteString(";\n")
			batchSize = 100
		}
	}

	w, err := os.Create(countryName + ".sql")
	if err != nil {
		return errors.WithStack(err)
	}
	defer w.Close()
	w.Write(buf.Bytes())

	return nil
}

func FormatData() error {
	f, err := os.Open("can.csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	if err := WriteCsvFile("can_with_code.csv", func(w *csv.Writer) error {
		w.Write([]string{"province", "province_code", "city", "city_code"})

		i := 0
		prevProvince := ""
		for {
			data, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}

				return errors.WithStack(err)
			}

			province, city := data[0], data[1]

			if prevProvince != province {
				i = 0
			}

			provinceCode := plProvince[data[0]]
			cityCode := fmt.Sprintf("%s-%d", provinceCode, i)

			w.Write([]string{province, provinceCode, city, cityCode})

			prevProvince = province

			i++
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func FormatJsonData() error {
	f, err := os.Open("can_area.json")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.WithStack(err)
	}

	var data []Area
	if err := json.Unmarshal(bs, &data); err != nil {
		return errors.WithStack(err)
	}

	if err := WriteCsvFile("can.csv", func(w *csv.Writer) error {
		w.Write([]string{"province", "city"})

		for _, item := range data {
			w.Write([]string{item.Parent, item.Name})
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func FetchAllCity() error {
	c := colly.NewCollector(
		colly.URLFilters(
			regexp.MustCompile(`https://can\.youbianku\.com/zh-hans/city-text`),
		),
		colly.Async(true),
	)

	var m sync.Map

	c.OnHTML(".view-city-text table a[href]", func(h *colly.HTMLElement) {
		link := h.Attr("href")
		fmt.Println("==> ", link, h.Text)

		data := strings.Split(link, "/")
		text := data[len(data)-1]
		fmt.Println("-->", text)

		text, _ = url.PathUnescape(text)

		d := strings.Split(text, ",")

		m.Store(link, Area{
			Parent: d[1],
			Name:   h.Text,
			Link:   link,
		})
	})

	c.OnHTML(".view-city-text li.pager-next", func(h *colly.HTMLElement) {
		link := h.ChildAttr("a[href]", "href")

		if link != "" {
			c.Visit(h.Request.AbsoluteURL(link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.Visit("https://can.youbianku.com/zh-hans/city-text")

	c.Wait()

	data := make([]Area, 0)
	m.Range(func(key, value any) bool {
		data = append(data, value.(Area))
		return true
	})

	bs, _ := json.Marshal(data)
	// fmt.Println(string(bs))

	f, err := os.Create("can_area.json")
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(bs)
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

	if err := WriteCsvFile("can.csv", func(w *csv.Writer) error {
		w.Write([]string{"province", "city", "area", "zip"})

		flushTimes := 50
		for data := range ch {
			w.Write(data)

			if flushTimes <= 0 {
				w.Flush()
				flushTimes = 50
			}

			flushTimes -= 1
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
		bs, _ := ioutil.ReadFile("can_area.json")
		var urls []Area
		json.Unmarshal(bs, &urls)

		fmt.Printf("total urls: %d\n", len(urls))

		for _, url := range urls {
			q.AddURL(fmt.Sprintf("https://can.youbianku.com%s", url.Link))
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

		c.OnHTML(".view-city-postcode", func(h *colly.HTMLElement) {
			path := h.Request.URL.Path
			data := strings.Split(path, "/")
			text, _ := url.PathUnescape(data[len(data)-1])
			d := strings.Split(text, ",")

			var hasData bool
			h.DOM.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
				hasData = true
				city := strings.TrimSpace(s.Find("td.views-field-field-province").Text())
				area := strings.TrimSpace(s.Find("td.views-field-field-city").Text())
				zip := strings.TrimSpace(s.Find("td.views-field-title a[href]").Text())
				ch <- []string{d[1], city, area, zip}
			})

			link, exist := h.DOM.Find("ul.pager li.pager-next.last a[href]").Attr("href")
			if exist && link != "" && hasData {
				c.Visit(h.Request.AbsoluteURL(link))
			}

		})

		// c.OnHTML(".view-city-postcode ul.pager li.pager-next.last a[href]", func(h *colly.HTMLElement) {
		// 	link := h.Attr("href")
		// 	if link != "" {
		// 		c.Visit(h.Request.AbsoluteURL(link))
		// 	}
		// })

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
