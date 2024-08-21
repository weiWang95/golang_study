package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.shoplazza.site/common/shoplazza-common/xid"
)

var countryName = "au"
var countryCode = "AU"
var ProvinceMap = map[string]string{
	"Australian Capital Territory": "AU-ACT",
	"New South Wales":              "AU-NSW",
	"Northern Territory":           "AU-NT",
	"Queensland":                   "AU-QLD",
	"South Australia":              "AU-SA",
	"Tasmania":                     "AU-TAS",
	"Victoria":                     "AU-VIC",
	"Western Australia":            "AU-WA",
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
	f, err := os.Open(countryName + ".csv")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read()

	if err := WriteCsvFile(countryName+"_with_code.csv", func(w *csv.Writer) error {
		w.Write([]string{"province", "province_code", "city", "city_code", "area", "area_code"})

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

			province, city, area := data[0], data[1], data[2]
			provinceCode := ProvinceMap[province]

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

			w.Write([]string{province, provinceCode, city, cityCode, area, areaCode})

			prevProvinceCode, prevCity, prevArea = provinceCode, city, area
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func FetchAllCity() error {
	c := colly.NewCollector(
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

	if err := WriteCsvFile(fmt.Sprintf("%s.csv", countryName), func(w *csv.Writer) error {
		w.Write([]string{"province", "city", "link"})

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

		q, _ := queue.New(2, &queue.InMemoryQueueStorage{MaxSize: 10000})
		// bs, _ := ioutil.ReadFile("can_area.json")
		// var urls []Area
		// json.Unmarshal(bs, &urls)

		// fmt.Printf("total urls: %d\n", len(urls))

		// for _, url := range urls {
		// 	q.AddURL(fmt.Sprintf("https://can.youbianku.com%s", url.Link))
		// }

		// for name, _ := range ProvinceMap {
		// 	q.AddURL(fmt.Sprintf("https://aus.postcodebase.com/zh-hans/region1/%s", strings.ReplaceAll(strings.ToLower(name), " ", "-")))
		// }

		// f, _ := os.Open("au_province.csv")
		// defer f.Close()
		// r := csv.NewReader(f)
		// r.LazyQuotes = true
		// r.Read()

		// for {
		// 	data, err := r.Read()
		// 	if err != nil {
		// 		if err == io.EOF {
		// 			break
		// 		}

		// 		panic(err)
		// 	}

		// 	q.AddURL(fmt.Sprintf("https://aus.postcodebase.com%s?p=%s", data[2], data[0]))
		// }
		q.AddURL("https://aus.postcodebase.com/zh-hans/region2/alexandrianew-south-wales?p=New South Wales")
		q.AddURL("https://aus.postcodebase.com/zh-hans/region2/burwoodnew-south-wales?p=New South Wales")

		c := colly.NewCollector(
		// colly.AllowURLRevisit(),
		)
		c.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: 5,
			RandomDelay: 2 * time.Second,
		})
		// c.WithTransport(&http.Transport{
		// 	DisableKeepAlives: true,
		// })
		extensions.RandomUserAgent(c)

		rp, err := proxy.RoundRobinProxySwitcher(ips...)
		if err != nil {
			panic(err)
		}
		c.SetProxyFunc(rp)

		c.OnRequest(func(r *colly.Request) {
			logrus.Infof("Visiting %s, proxy:%s", r.URL.String(), r.ProxyURL)
		})

		c.OnResponse(func(r *colly.Response) {
			logrus.Infof("Visit success %s, proxy:%s", r.Request.URL.String(), r.Request.ProxyURL)
		})

		c.OnError(func(r *colly.Response, err error) {
			fmt.Printf("Error: Code:%d proxy:%s Err:%v\n", r.StatusCode, r.Request.ProxyURL, err)
			ech <- r.Request.URL.String()
			// r.Request.Retry()

			// nr, err := r.Request.New("GET", r.Request.URL.String(), nil)
			// if err != nil {
			// 	fmt.Printf("err -> %v\n", err)
			// } else {
			// 	q.AddRequest(nr)
			// }
		})

		c.OnHTML(".view-region2-region3", func(h *colly.HTMLElement) {
			q := h.Request.URL.Query()
			parent := q.Get("p")
			var hasData bool
			h.DOM.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
				hasData = true
				province := strings.TrimSpace(s.Find("td.views-field-field-region2-text").Text())
				link := s.Find("td.views-field-field-region3-text a[href]")
				city := strings.TrimSpace(link.Text())
				// href, _ := link.Attr("href")
				ch <- []string{parent, province, city}
			})

			link, exist := h.DOM.Find("ul.pager li.pager-next.last a[href]").Attr("href")
			if exist && link != "" && hasData {
				url := h.Request.AbsoluteURL(link)
				// fmt.Println("next -> ", url)
				c.Visit(url)
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

var ips = []string{
	"http://43.157.122.250:19424",
	"http://49.51.189.174:19483",
	"http://49.51.189.174:19480",
	"http://43.128.114.47:14807",
	"http://49.51.189.174:19485",
	"http://43.157.122.250:19395",
	"http://43.157.122.250:19397",
	"http://43.157.122.250:19421",
	"http://49.51.189.174:19495",
	"http://43.128.114.47:14810",
	"http://49.51.189.174:19479",
	"http://49.51.189.174:19460",
	"http://43.157.122.250:19428",
	"http://49.51.189.174:19472",
	"http://43.128.114.47:14822",
	"http://43.157.122.250:19425",
	"http://49.51.189.174:19494",
	"http://49.51.189.174:19468",
	"http://49.51.189.174:19481",
	"http://49.51.189.174:19484",
	"http://49.51.189.174:19489",
	"http://43.157.122.250:19394",
	"http://43.157.122.250:19399",
	"http://43.157.122.250:19406",
	"http://43.128.114.47:14813",
	"http://43.157.122.250:19416",
	"http://49.51.189.174:19492",
	"http://49.51.189.174:19491",
	"http://43.157.122.250:19404",
	"http://43.157.122.250:19405",
	"http://49.51.189.174:19488",
	"http://43.157.122.250:19398",
	"http://43.128.114.47:14819",
	"http://43.157.122.250:19429",
	"http://49.51.189.174:19458",
	"http://49.51.189.174:19496",
	"http://43.157.122.250:19426",
	"http://49.51.189.174:19457",
	"http://43.157.122.250:19409",
	"http://43.157.122.250:19393",
	"http://43.128.114.47:14814",
	"http://49.51.189.174:19493",
	"http://49.51.189.174:19490",
	"http://43.128.114.47:14811",
	"http://43.157.122.250:19392",
	"http://43.157.122.250:19403",
	"http://43.157.122.250:19422",
	"http://43.157.122.250:19418",
	"http://43.128.114.47:14812",
	"http://49.51.189.174:19466",
	"http://43.157.122.250:19415",
	"http://43.157.122.250:19407",
	"http://49.51.189.174:19486",
	"http://43.128.114.47:14808",
	"http://49.51.189.174:19462",
	"http://49.51.189.174:19482",
	"http://49.51.189.174:19464",
	"http://43.157.122.250:19396",
	"http://43.157.122.250:19430",
	"http://49.51.189.174:19461",
	"http://43.128.114.47:14823",
	"http://43.157.122.250:19423",
	"http://43.157.122.250:19431",
	"http://49.51.189.174:19471",
	"http://43.128.114.47:14824",
	"http://43.157.122.250:19420",
	"http://49.51.189.174:19463",
	"http://49.51.189.174:19465",
	"http://49.51.189.174:19469",
	"http://43.128.114.47:14815",
	"http://43.157.122.250:19419",
	"http://43.128.114.47:14816",
	"http://49.51.189.174:19474",
	"http://43.157.122.250:19414",
	"http://43.157.122.250:19400",
	"http://43.157.122.250:19412",
	"http://43.128.114.47:14826",
	"http://49.51.189.174:19476",
	"http://43.128.114.47:14825",
	"http://43.128.114.47:14818",
	"http://49.51.189.174:19459",
	"http://43.157.122.250:19417",
	"http://49.51.189.174:19470",
	"http://43.128.114.47:14817",
	"http://49.51.189.174:19467",
	"http://49.51.189.174:19477",
	"http://49.51.189.174:19473",
	"http://43.157.122.250:19410",
	"http://43.157.122.250:19408",
	"http://49.51.189.174:19478",
	"http://43.128.114.47:14820",
	"http://43.128.114.47:14809",
	"http://43.157.122.250:19413",
	"http://43.157.122.250:19427",
	"http://49.51.189.174:19475",
	"http://43.157.122.250:19402",
	"http://43.157.122.250:19401",
	"http://49.51.189.174:19487",
	"http://43.128.114.47:14821",
	"http://43.157.122.250:19411",
}
