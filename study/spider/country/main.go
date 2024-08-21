package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var allLangs = []string{
	"en", "zh", "zh-tw", "fr", "de", "ru", "pt", "es", "ko", "ja", "ar",
}
var codes = "AL,AS,AD,AO,AI,AQ,AG,AR,AM,AW,AU,AT,AZ,BH,BD,BY,BO,BR,BG,BF,CA,CN,KM,CR,CI,CU,CY,DK,DJ,DO,EC,SZ,FJ,GA,GM,DE,GL,GG,HK,IS,IN,ID,IM,IT,JP,JE,KI,KP,KR,LS,LU,MO,MG,MY,YT,NE,NO,PA,PL,RU,RW,ST,RS,SL,SG,ZA,ES,SE,CH,TW,TG,TN,TR,TM,UA,AE,GB,US,VE,ZM,ZW"
var codes2 = "AE,AR,AU,BH,BR,CA,CH,CL,CN,CO,DE,EG,ES,FR,GB,GH,GR,HK,HU,ID,IL,IN,IQ,IR,IT,JP,KR,KW,LT,MA,MO,MX,MY,NG,NL,NO,NZ,OM,PE,PH,PL,PT,QA,RO,RS,RU,SA,SE,SG,SS,TH,TR,TW,UA,US,VN,ZA"
var codes3 = "KR,GB,HK,ID,IN,IT,JP,MO,MY,NO,PL,RS,RU,SE,SG,TR,TW,UA,US,ZA"

var langMap = map[string]string{
	"zh-TW": "zh-tw",
}

type Lang struct {
	Code string `json:"code"`
}

func main() {
	// langs, err := AllLangs()
	// if err != nil {
	// 	panic(err)
	// }

	// for _, lang := range langs {
	// 	code := lang.Code
	// 	if l, ok := langMap[code]; ok {
	// 		code = l
	// 	} else {
	// 		code = code[0:2]
	// 	}

	// 	if err := StartSpider(code); err != nil {
	// 		logrus.WithError(err).Errorf("%s fail", lang.Code)
	// 	}

	// 	time.Sleep(3 * time.Second)
	// }

	// if err := GroupData(context.TODO(), codes3[0:2]); err != nil {
	// 	panic(err)
	// }
	// return

	for _, code := range strings.Split(codes3, ",") {
		for _, lang := range allLangs {
			if err := StartSpider(lang, code); err != nil {
				logrus.WithError(err).Errorf("%s %s fail", lang, code)
			}
			// return
			time.Sleep(3 * time.Second)
		}
		return
	}

	// if err := StartSpider("ja"); err != nil {
	// 	logrus.WithError(err).Errorf("es fail")
	// }
}

func AllLangs() ([]Lang, error) {
	bs, err := ioutil.ReadFile("lang.json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var res []Lang
	if err := json.Unmarshal(bs, &res); err != nil {
		return nil, errors.WithStack(err)
	}
	return res, nil
}

func StartSpider(lang, code string) error {
	ch, ech := fetchData(context.TODO(), lang, code)

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

	os.Mkdir(code, os.ModePerm)

	if err := WriteCsvFile(fmt.Sprintf("%s/%s.csv", code, lang), func(w *csv.Writer) error {
		// w.Write([]string{"code", "name"})

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

func fetchData(ctx context.Context, lang, code string) (chan []string, chan string) {
	ch := make(chan []string)
	ech := make(chan string)

	go func() {
		defer func() {
			close(ch)
			close(ech)
		}()

		q, _ := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: 1000})
		// bs, _ := ioutil.ReadFile(countryName + "_area.json")
		// var urls []Area
		// json.Unmarshal(bs, &urls)

		// fmt.Printf("total urls: %d\n", len(urls))

		// for _, url := range urls {
		// 	q.AddURL(fmt.Sprintf("https://kor.postcodebase.com%s", url.Link))
		// }

		// for name, _ := range ProvinceMap {
		// 	q.AddURL(fmt.Sprintf("https://aus.postcodebase.com/zh-hans/region1/%s", strings.ReplaceAll(strings.ToLower(name), " ", "-")))
		// }
		q.AddURL(fmt.Sprintf("https://%s.wikipedia.org/wiki/ISO_3166-2:%s", lang, code))
		// q.AddURL("https://ja.wikipedia.org/wiki/ISO_3166-2:US")

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
		// 	panic(err)
		// }
		// c.SetProxyFunc(rp)

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

		c.OnHTML("#mw-content-text", func(h *colly.HTMLElement) {
			h.DOM.Find(".mw-parser-output > table tbody tr").Each(func(i int, s *goquery.Selection) {
				if s.HasClass("sortbottom") {
					return
				}
				if _, exist := s.Attr("class"); exist {
					return
				}

				d := s.Find("td")
				if len(d.Nodes) == 1 {
					return
				}
				data := make([]string, 0)

				d.Each(func(j int, s2 *goquery.Selection) {
					if s2.Find("a").Length() >= 1 {
						t := s2.Find("a").Last().Text()
						if strings.HasPrefix(t, "[") {
							data = append(data, s2.Text())
						} else {
							data = append(data, t)
						}
					} else {
						data = append(data, s2.Text())
					}
				})
				name := s.Find("td span.flagicon").Next().Text()
				name2, _ := s.Find("td span a.mw-file-description").Attr("title")
				data = append(data, name, name2)

				ch <- data

				// // name := strings.TrimSpace(s.Find("td:first-child a[href]").Text())
				// code := strings.TrimSpace(s.Find("td:nth-child(5) code").Text())
				// ch <- []string{name, code}
			})

			// link, exist := h.DOM.Find("ul.pager li.pager-next.last a[href]").Attr("href")
			// if exist && link != "" && hasData {
			// 	url := h.Request.AbsoluteURL(link)
			// 	fmt.Println("next -> ", url)
			// 	// c.Visit(url)
			// 	q.AddURL(url)
			// }
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

func GroupData(context context.Context, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return errors.WithStack(err)
	}

	data := make(map[string]map[string]string)
	for _, item := range files {
		if strings.HasPrefix(item.Name(), "all") {
			continue
		}
		if !strings.HasSuffix(item.Name(), ".csv") {
			continue
		}
		lang := strings.TrimSuffix(item.Name(), ".csv")

		if err := EachCsvFile(fmt.Sprintf("%s/%s", dir, item.Name()), func(d []string) {
			if len(d) != 2 {
				return
			}
			code, value := trimStr(d[0]), trimStr(d[1])
			if _, ok := data[code]; !ok {
				data[code] = make(map[string]string)
			}
			// fmt.Printf("%s %s -> %s\n", code, lang, value)
			data[code][lang] = value
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := WriteCsvFile(fmt.Sprintf("%s/all.csv", dir), func(w *csv.Writer) error {
		w.Write(append([]string{"Code"}, allLangs...))
		tmp := make([]string, 0, len(allLangs)+1)
		for code, item := range data {
			tmp = append(tmp, code)
			for _, lang := range allLangs {
				tmp = append(tmp, item[lang])
			}
			w.Write(tmp)
			tmp = tmp[0:0]
		}
		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func EachCsvFile(filename string, fn func(data []string)) error {
	f, _ := os.Open(filename)
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true

	for {
		data, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		fn(data)
	}
	return nil
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

func trimStr(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", ""))
}
