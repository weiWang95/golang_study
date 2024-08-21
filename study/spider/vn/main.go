package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

var countryName = "vnm"
var countryCode = "VN"
var ProvinceMap = map[string]string{
	"An Giang":          "VN-44",
	"Bà Rịa - Vũng Tàu": "VN-43",
	"Bắc Giang":         "VN-54",
	"Bắc Kạn":           "VN-53",
	"Bạc Liêu":          "VN-55",
	"Bắc Ninh":          "VN-56",
	"Bến Tre":           "VN-50",
	"Bình Định":         "VN-31",
	"Bình Dương":        "VN-57",
	"Bình Phước":        "VN-58",
	"Bình Thuận":        "VN-40",
	"Cà Mau":            "VN-59",
	"Cao Bằng":          "VN-04",
	"Đắk Lắk":           "VN-33",
	"Đắk Nông":          "VN-72",
	"Điện Biên":         "VN-71",
	"Đồng Nai":          "VN-39",
	"Đồng Tháp":         "VN-45",
	"Gia Lai":           "VN-30",
	"Hà Giang":          "VN-03",
	"Hà Nam":            "VN-63",
	"Hà Tĩnh":           "VN-23",
	"Hải Dương":         "VN-61",
	"Hậu Giang":         "VN-73",
	"Hòa Bình":          "VN-14",
	"Hưng Yên":          "VN-66",
	"Khánh Hòa":         "VN-34",
	"Kiên Giang":        "VN-47",
	"Kon Tum":           "VN-28",
	"Lai Châu":          "VN-01",
	"Lâm Đồng":          "VN-35",
	"Lạng Sơn":          "VN-09",
	"Lào Cai":           "VN-02",
	"Long An":           "VN-41",
	"Nam Định":          "VN-67",
	"Nghệ An":           "VN-22",
	"Ninh Bình":         "VN-18",
	"Ninh Thuận":        "VN-36",
	"Phú Thọ":           "VN-68",
	"Phú Yên":           "VN-32",
	"Quảng Bình":        "VN-24",
	"Quảng Nam":         "VN-27",
	"Quảng Ngãi":        "VN-29",
	"Quảng Ninh":        "VN-13",
	"Quảng Trị":         "VN-25",
	"Sóc Trăng":         "VN-52",
	"Sơn La":            "VN-05",
	"Tây Ninh":          "VN-37",
	"Thái Bình":         "VN-20",
	"Thái Nguyên":       "VN-69",
	"Thanh Hóa":         "VN-21",
	"Thừa Thiên - Huế":  "VN-26",
	"Tiền Giang":        "VN-46",
	"Trà Vinh":          "VN-51",
	"Tuyên Quang":       "VN-07",
	"Vĩnh Long":         "VN-49",
	"Vĩnh Phúc":         "VN-70",
	"Yên Bái":           "VN-06",
	"Cần Thơ":           "VN-CT",
	"Đà Nẵng":           "VN-DN",
	"Hà Nội":            "VN-HN",
	"Hải Phòng":         "VN-HP",
	"Hồ Chí Minh":       "VN-SG",
}

type Area struct {
	Code string `json:"code,omitempty"`
	Name string `json:"name,omitempty"`
	Link string `json:"link,omitempty"`
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

			province, city := data[0], data[2]

			if prevProvince != province {
				i = 0
			}

			provinceCode, ok := ProvinceMap[data[0]]
			if !ok {
				fmt.Println("no match code: ", data[0])
			}
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

func FetchAllCity() error {
	c := colly.NewCollector(
		colly.Async(true),
	)

	var m sync.Map

	c.OnHTML(".view-region2-text", func(h *colly.HTMLElement) {
		h.DOM.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
			// provinceCode := strings.TrimSpace(s.Find("td.views-field-field-uf").Text())
			link := s.Find("td.views-field-title a[href]")
			city := strings.TrimSpace(link.Text())
			href, _ := link.Attr("href")

			m.Store(href, Area{
				Name: city,
				Link: href,
			})
		})
	})

	c.OnHTML(".view-region2-text ul.pager li.pager-next", func(h *colly.HTMLElement) {
		link := h.ChildAttr("a[href]", "href")

		if link != "" {
			c.Visit(h.Request.AbsoluteURL(link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.Visit("https://vnm.postcodebase.com/zh-hans/region2-text")

	c.Wait()

	data := make([]Area, 0)
	m.Range(func(key, value any) bool {
		data = append(data, value.(Area))
		return true
	})

	bs, _ := json.Marshal(data)
	// fmt.Println(string(bs))

	f, err := os.Create(countryName + "_area.json")
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
		w.Write([]string{"province", "provinceCode", "city", "link"})

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

		q, _ := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: 100})

		// bs, _ := ioutil.ReadFile(countryName + "_area.json")
		// var urls []Area
		// json.Unmarshal(bs, &urls)
		// fmt.Printf("total urls: %d\n", len(urls))
		// for _, url := range urls {
		// 	q.AddURL(fmt.Sprintf("https://vnm.postcodebase.com%s?p=%s", url.Link, url.Name))
		// }

		bs, _ := ioutil.ReadFile("fail_urls.json")
		var urls []string
		json.Unmarshal(bs, &urls)
		for _, url := range urls {
			q.AddURL(url)
		}

		// for name, _ := range ProvinceMap {
		// 	q.AddURL(fmt.Sprintf("https://aus.postcodebase.com/zh-hans/region1/%s", strings.ReplaceAll(strings.ToLower(name), " ", "-")))
		// }

		// f, _ := os.Open(countryName + "_city.csv")
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

		// 	q.AddURL(fmt.Sprintf("https://vnm.postcodebase.com%s?p=%s", data[3], data[0], data[1]))
		// }

		c := colly.NewCollector(
		// colly.AllowURLRevisit(),
		)
		c.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: 1,
			RandomDelay: 10 * time.Second,
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
			query := h.Request.URL.Query()
			province := query.Get("p")

			var hasData bool
			h.DOM.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
				hasData = true
				city := strings.TrimSpace(s.Find("td.views-field-field-region2-text").Text())
				link := s.Find("td.views-field-title a[href]")
				area := strings.TrimSpace(link.Text())
				// href, _ := link.Attr("href")
				ch <- []string{province, city, area}
			})

			link, exist := h.DOM.Find("ul.pager li.pager-next.last a[href]").Attr("href")
			if exist && link != "" && hasData {
				url := h.Request.AbsoluteURL(link)
				// fmt.Println("next -> ", url)
				// c.Visit(url)

				q.AddURL(url)
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
	"http://49.51.189.174:19726",
	"http://43.157.122.250:19672",
	"http://49.51.189.174:19710",
	"http://43.157.122.250:19661",
	"http://43.157.122.250:19652",
	"http://49.51.189.174:19718",
	"http://43.157.122.250:19666",
	"http://43.157.122.250:19651",
	"http://43.157.122.250:19656",
	"http://43.157.122.250:19647",
	"http://49.51.189.174:19733",
	"http://43.128.114.47:14808",
	"http://43.157.122.250:19657",
	"http://43.157.122.250:19645",
	"http://49.51.189.174:19715",
	"http://49.51.189.174:19717",
	"http://43.157.122.250:19669",
	"http://43.128.114.47:14799",
	"http://43.157.122.250:19668",
	"http://49.51.189.174:19708",
	"http://49.51.189.174:19713",
	"http://43.128.114.47:14794",
	"http://49.51.189.174:19741",
	"http://49.51.189.174:19724",
	"http://43.128.114.47:14798",
	"http://43.157.122.250:19662",
	"http://49.51.189.174:19731",
	"http://43.157.122.250:19659",
	"http://43.157.122.250:19649",
	"http://43.128.114.47:14803",
	"http://43.128.114.47:14801",
	"http://43.128.114.47:14806",
	"http://43.157.122.250:19675",
	"http://43.128.114.47:14793",
	"http://43.157.122.250:19655",
	"http://49.51.189.174:19709",
	"http://43.157.122.250:19653",
	"http://49.51.189.174:19737",
	"http://43.157.122.250:19667",
	"http://49.51.189.174:19732",
	"http://43.157.122.250:19680",
	"http://49.51.189.174:19745",
	"http://43.157.122.250:19646",
	"http://43.128.114.47:14807",
	"http://43.128.114.47:14810",
	"http://43.128.114.47:14796",
	"http://49.51.189.174:19738",
	"http://49.51.189.174:19712",
	"http://43.157.122.250:19660",
	"http://49.51.189.174:19722",
	"http://49.51.189.174:19743",
	"http://43.128.114.47:14802",
	"http://43.157.122.250:19658",
	"http://49.51.189.174:19727",
	"http://43.157.122.250:19674",
	"http://43.157.122.250:19673",
	"http://43.157.122.250:19679",
	"http://43.128.114.47:14800",
	"http://49.51.189.174:19723",
	"http://49.51.189.174:19719",
	"http://43.157.122.250:19644",
	"http://49.51.189.174:19714",
	"http://49.51.189.174:19740",
	"http://49.51.189.174:19729",
	"http://43.157.122.250:19650",
	"http://49.51.189.174:19706",
	"http://49.51.189.174:19734",
	"http://49.51.189.174:19725",
	"http://49.51.189.174:19730",
	"http://43.128.114.47:14809",
	"http://43.128.114.47:14811",
	"http://43.157.122.250:19665",
	"http://49.51.189.174:19736",
	"http://49.51.189.174:19739",
	"http://43.157.122.250:19671",
	"http://43.157.122.250:19664",
	"http://49.51.189.174:19728",
	"http://43.157.122.250:19654",
	"http://43.128.114.47:14797",
	"http://49.51.189.174:19742",
	"http://49.51.189.174:19711",
	"http://43.157.122.250:19643",
	"http://43.128.114.47:14804",
	"http://43.157.122.250:19648",
	"http://43.157.122.250:19670",
	"http://49.51.189.174:19716",
	"http://43.157.122.250:19678",
	"http://43.128.114.47:14812",
	"http://49.51.189.174:19707",
	"http://43.157.122.250:19663",
	"http://43.128.114.47:14805",
	"http://43.157.122.250:19677",
	"http://43.157.122.250:19681",
	"http://43.128.114.47:14795",
	"http://43.157.122.250:19682",
	"http://43.157.122.250:19676",
	"http://49.51.189.174:19720",
	"http://49.51.189.174:19721",
	"http://49.51.189.174:19744",
	"http://49.51.189.174:19735",
}
