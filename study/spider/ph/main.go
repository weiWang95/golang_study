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

var countryName = "phl"
var countryCode = "PH"
var ProvinceMap = map[string]string{
	"Abra":                "PH-ABR",
	"Agusan del Norte":    "PH-AGN",
	"Agusan del Sur":      "PH-AGS",
	"Aklan":               "PH-AKL",
	"Albay":               "PH-ALB",
	"Antique":             "PH-ANT",
	"Apayao":              "PH-APA",
	"Aurora":              "PH-AUR",
	"Basilan":             "PH-BAS",
	"Bataan":              "PH-BAN",
	"Batanes":             "PH-BTN",
	"Batangas":            "PH-BTG",
	"Benguet":             "PH-BEN",
	"Biliran":             "PH-BIL",
	"Bohol":               "PH-BOH",
	"Bukidnon":            "PH-BUK",
	"Bulacan":             "PH-BUL",
	"Cagayan":             "PH-CAG",
	"Camarines Norte":     "PH-CAN",
	"Camarines Sur":       "PH-CAS",
	"Camiguin":            "PH-CAM",
	"Capiz":               "PH-CAP",
	"Catanduanes":         "PH-CAT",
	"Cavite":              "PH-CAV",
	"Cebu":                "PH-CEB",
	"Cotabato":            "PH-NCO",
	"Davao Occidental":    "PH-DVO",
	"Davao Oriental":      "PH-DAO",
	"Compostela Valley":   "PH-COM",
	"Davao del Norte":     "PH-DAV",
	"Davao del Sur":       "PH-DAS",
	"Dinagat Islands":     "PH-DIN",
	"Eastern Samar":       "PH-EAS",
	"Guimaras":            "PH-GUI",
	"Ifugao":              "PH-IFU",
	"Ilocos Norte":        "PH-ILN",
	"Ilocos Sur":          "PH-ILS",
	"Iloilo":              "PH-ILI",
	"Isabela":             "PH-ISA",
	"Kalinga":             "PH-KAL",
	"La Union":            "PH-LUN",
	"Laguna":              "PH-LAG",
	"Lanao del Norte":     "PH-LAN",
	"Lanao del Sur":       "PH-LAS",
	"Leyte":               "PH-LEY",
	"Maguindanao":         "PH-MAG",
	"Marinduque":          "PH-MAD",
	"Masbate":             "PH-MAS",
	"Metro Manila":        "PH-00",
	"Misamis Occidental":  "PH-MSC",
	"Misamis Oriental":    "PH-MSR",
	"Mountain Province":   "PH-MOU",
	"Negros Occidental":   "PH-NEC",
	"Negros Oriental":     "PH-NER",
	"Northern Samar":      "PH-NSA",
	"Nueva Ecija":         "PH-NUE",
	"Nueva Vizcaya":       "PH-NUV",
	"Occidental Mindoro":  "PH-MDC",
	"Oriental Mindoro":    "PH-MDR",
	"Palawan":             "PH-PLW",
	"Pampanga":            "PH-PAM",
	"Pangasinan":          "PH-PAN",
	"Quezon":              "PH-QUE",
	"Quirino":             "PH-QUI",
	"Rizal":               "PH-RIZ",
	"Romblon":             "PH-ROM",
	"Samar":               "PH-WSA",
	"Sarangani":           "PH-SAR",
	"Siquijor":            "PH-SIG",
	"Sorsogon":            "PH-SOR",
	"South Cotabato":      "PH-SCO",
	"Southern Leyte":      "PH-SLE",
	"Sultan Kudarat":      "PH-SUK",
	"Sulu":                "PH-SLU",
	"Surigao del Norte":   "PH-SUN",
	"Surigao del Sur":     "PH-SUR",
	"Tarlac":              "PH-TAR",
	"Tawi-Tawi":           "PH-TAW",
	"Zambales":            "PH-ZMB",
	"Zamboanga Sibugay":   "PH-ZSI",
	"Zamboanga del Norte": "PH-ZAN",
	"Zamboanga del Sur":   "PH-ZAS",
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

		province, provinceCode, city, cityCode, area, areaCode, zip := data[0], data[1], data[2], data[3], "", "", ""
		if len(data) >= 5 {
			area, areaCode = data[4], data[5]
		}
		if len(data) >= 6 {
			zip = data[6]
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
			buf.WriteString("INSERT INTO `customization_area`(`id`,`store_id`,`parent_code`,`area_code`,`area_name`,`zip`) VALUES\n")
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf(`(%d,'%s','%s','%s',"%s",'%s')`, xid.Get(), "0", provinceCode, cityCode, city, zip))
				if areaCode != "" {
					buf.WriteRune(',')
				}
				buf.WriteRune('\n')
				prevCityCode = cityCode
			}
			if areaCode != "" {
				buf.WriteString(fmt.Sprintf(`(%d,'%s','%s','%s',"%s",'%s')`, xid.Get(), "0", cityCode, areaCode, area, zip))
				buf.WriteRune('\n')
			}
		} else {
			if prevCityCode != cityCode {
				buf.WriteString(fmt.Sprintf(`,(%d,'%s','%s','%s',"%s",'%s')`, xid.Get(), "0", provinceCode, cityCode, city, zip))
				buf.WriteRune('\n')
				prevCityCode = cityCode
			}
			if areaCode != "" {
				buf.WriteString(fmt.Sprintf(`,(%d,'%s','%s','%s',"%s",'%s')`, xid.Get(), "0", cityCode, areaCode, area, zip))
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

	arr := make(map[string]interface{})

	if err := WriteCsvFile(countryName+"_with_code.csv", func(w *csv.Writer) error {
		w.Write([]string{"province", "province_code", "city", "city_code", "area", "area_code", "zip"})

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

			province, city, area, zip := data[0], data[1], data[2], data[3]
			provinceCode, ok := ProvinceMap[province]
			if !ok {
				arr[province] = nil
			}

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

			w.Write([]string{province, provinceCode, city, cityCode, area, areaCode, zip})

			prevProvinceCode, prevCity, prevArea = provinceCode, city, area
		}

		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	fmt.Println("not matched ", arr)

	return nil
}

func FetchAllCity() error {
	c := colly.NewCollector(
		colly.Async(true),
	)
	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	var m sync.Map

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error: Code:%d proxy:%s Err:%v\n", r.StatusCode, r.Request.ProxyURL, err)
	})

	c.OnHTML(".view-region2-text .views-field-title a[href]", func(h *colly.HTMLElement) {
		link := h.Attr("href")
		fmt.Println("==> ", link, h.Text)

		m.Store(link, Area{
			Name: h.Text,
			Link: link,
		})
	})

	c.OnHTML(".view-region2-text li.pager-next", func(h *colly.HTMLElement) {
		link := h.ChildAttr("a[href]", "href")

		if link != "" {
			c.Visit(h.Request.AbsoluteURL(link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	c.OnResponse(func(r *colly.Response) {
		// fmt.Printf("response: %v\n%s\n", r.StatusCode, string(r.Body))
	})

	c.Visit("https://phl.postcodebase.com/zh-hans/region2-text")

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

		q, _ := queue.New(5, &queue.InMemoryQueueStorage{MaxSize: 1000})
		// bs, _ := ioutil.ReadFile(countryName + "_area.json")
		// var urls []Area
		// json.Unmarshal(bs, &urls)

		// fmt.Printf("total urls: %d\n", len(urls))

		// for _, url := range urls {
		// 	q.AddURL(fmt.Sprintf("https://phl.postcodebase.com%s", url.Link))
		// }

		// for name, _ := range ProvinceMap {
		// 	q.AddURL(fmt.Sprintf("https://aus.postcodebase.com/zh-hans/region1/%s", strings.ReplaceAll(strings.ToLower(name), " ", "-")))
		// }

		f, _ := os.Open(countryName + "_city.csv")
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

				panic(err)
			}

			q.AddURL(fmt.Sprintf("https://phl.postcodebase.com%s?p=%s", data[2], data[0]))
		}

		c := colly.NewCollector(
		// colly.AllowURLRevisit(),
		)
		c.Limit(&colly.LimitRule{
			DomainGlob:  "*",
			Parallelism: 5,
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
			logrus.Infof("Visit success:%d %s, proxy:%s", r.StatusCode, r.Request.URL.String(), r.Request.ProxyURL)
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

		c.OnHTML(".view-test1", func(h *colly.HTMLElement) {
			query := h.Request.URL.Query()
			p := query.Get("p")
			var hasData bool
			h.DOM.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
				hasData = true
				city := strings.TrimSpace(s.Find("td.views-field-field-region3").Text())
				area := strings.TrimSpace(s.Find("td.views-field-field-city").Text())
				link := s.Find("td.views-field-field-zip a[href]")
				zip := strings.TrimSpace(link.Text())
				// href, _ := link.Attr("href")
				ch <- []string{p, city, area, zip}
			})

			link, exist := h.DOM.Find("ul.pager li.pager-next.last a[href]").Attr("href")
			if exist && link != "" && hasData {
				url := h.Request.AbsoluteURL(link)
				fmt.Println("next -> ", url)
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
	"http://43.157.122.250:19195",
	"http://43.157.122.250:19227",
	"http://49.51.189.174:19379",
	"http://43.128.114.47:14992",
	"http://49.51.189.174:19370",
	"http://43.157.122.250:19203",
	"http://43.157.122.250:19212",
	"http://43.157.122.250:19229",
	"http://43.157.122.250:19208",
	"http://49.51.189.174:19395",
	"http://49.51.189.174:19391",
	"http://43.128.114.47:14980",
	"http://43.157.122.250:19200",
	"http://43.157.122.250:19210",
	"http://43.128.114.47:14984",
	"http://49.51.189.174:19393",
	"http://49.51.189.174:19376",
	"http://43.157.122.250:19201",
	"http://43.157.122.250:19214",
	"http://43.157.122.250:19196",
	"http://49.51.189.174:19362",
	"http://49.51.189.174:19384",
	"http://43.157.122.250:19211",
	"http://49.51.189.174:19375",
	"http://43.128.114.47:14988",
	"http://43.128.114.47:14978",
	"http://43.128.114.47:14983",
	"http://49.51.189.174:19371",
	"http://43.157.122.250:19219",
	"http://43.157.122.250:19199",
	"http://43.157.122.250:19209",
	"http://43.157.122.250:19218",
	"http://49.51.189.174:19385",
	"http://43.128.114.47:14989",
	"http://43.157.122.250:19204",
	"http://43.157.122.250:19198",
	"http://49.51.189.174:19387",
	"http://49.51.189.174:19390",
	"http://49.51.189.174:19383",
	"http://43.128.114.47:14979",
	"http://43.128.114.47:14985",
	"http://43.157.122.250:19226",
	"http://49.51.189.174:19358",
	"http://43.128.114.47:14977",
	"http://43.157.122.250:19206",
	"http://43.157.122.250:19231",
	"http://43.128.114.47:14993",
	"http://43.157.122.250:19215",
	"http://43.157.122.250:19216",
	"http://49.51.189.174:19394",
	"http://43.128.114.47:14991",
	"http://49.51.189.174:19374",
	"http://49.51.189.174:19368",
	"http://43.157.122.250:19205",
	"http://43.157.122.250:19194",
	"http://43.128.114.47:14990",
	"http://49.51.189.174:19357",
	"http://43.157.122.250:19228",
	"http://43.128.114.47:14982",
	"http://43.157.122.250:19207",
	"http://43.128.114.47:14987",
	"http://49.51.189.174:19365",
	"http://49.51.189.174:19359",
	"http://43.157.122.250:19217",
	"http://49.51.189.174:19380",
	"http://43.157.122.250:19193",
	"http://43.157.122.250:19230",
	"http://49.51.189.174:19386",
	"http://43.157.122.250:19202",
	"http://43.157.122.250:19221",
	"http://43.157.122.250:19197",
	"http://49.51.189.174:19369",
	"http://43.157.122.250:19223",
	"http://49.51.189.174:19361",
	"http://49.51.189.174:19364",
	"http://43.157.122.250:19225",
	"http://49.51.189.174:19356",
	"http://43.128.114.47:14994",
	"http://49.51.189.174:19360",
	"http://49.51.189.174:19389",
	"http://43.157.122.250:19192",
	"http://49.51.189.174:19377",
	"http://43.157.122.250:19220",
	"http://43.128.114.47:14986",
	"http://43.128.114.47:14981",
	"http://49.51.189.174:19381",
	"http://49.51.189.174:19363",
	"http://43.157.122.250:19222",
	"http://49.51.189.174:19372",
	"http://49.51.189.174:19366",
	"http://49.51.189.174:19382",
	"http://43.157.122.250:19213",
	"http://49.51.189.174:19378",
	"http://49.51.189.174:19367",
	"http://49.51.189.174:19388",
	"http://43.128.114.47:14995",
	"http://49.51.189.174:19373",
	"http://43.157.122.250:19224",
	"http://49.51.189.174:19392",
	"http://43.128.114.47:14996",
}
