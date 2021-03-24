package main

import (
	"fmt"
	"flag"
	"bytes"
	"regexp"
	"os"
	"winse.com/spider/http"
)

var (
	targetUrl = *flag.String("targetUrl", "https://www.zhihu.com/question/383670825", "spider target url")
	deep 			= *flag.Int("deep", 1, "spider deep")
)

func downloadImage(url string) {
	fmt.Println("downloadImage url -> ", url)

	re := regexp.MustCompile(`[\.\w\-\:\/]+\/([\w\-\.]+\.(\w+))`)
	data := re.FindAllStringSubmatch(url, -1)
	fileFullName := data[0][1]

	filePath := bytes.NewBuffer(nil)
	filePath.WriteString("tmp/")
	filePath.WriteString(fileFullName)

	os.MkdirAll("tmp", os.ModePerm)
	file, err := os.Create(filePath.String())
	if err != nil {
		fmt.Printf("Create File Failed -> %+v\n", err)
		return
	}

	defer file.Close()

	http.Download(url, file)

	file.Sync()

	fmt.Printf("Success -> %s\n", filePath.String())
}

func main() {
	flag.Parse()

	fmt.Printf("Spider Start! target:%s, deep: %d\n", targetUrl, deep)

	ch := make(chan string, 100)

	for i := 0; i < 4; i ++ {
		go func() {
			no := &i
			fmt.Printf("D%d Start!\n", no)
			for {
				url, ok := <- ch
				if !ok {
					break
				}

				fmt.Printf("D%d recv -> %s\n", no, url)

				downloadImage(url)
			}
			fmt.Printf("D%d end!\n", i)
		}()
	}


	res := http.Get(targetUrl)

	re := regexp.MustCompile(`https?:\/\/[\w\.\-\/]+\.(?:jpg|jpeg|png|gif)`)
	urls := re.FindAllStringSubmatch(res, -1)
	fmt.Println("Image Urls Count -> ", len(urls))

	for i := 0; i < len(urls); i ++ {
		ch <- urls[i][0]
	}

	fmt.Println("Spider Exit ~ ")
}