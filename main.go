package main

import (
	"fmt"
	"flag"
	"regexp"
	"winse.com/spider/http"
)

var (
	targetUrl = *flag.String("targetUrl", "https://zhuanlan.zhihu.com/p/341554869", "spider target url")
	deep 			= *flag.Int("deep", 1, "spider deep")
	basePath  = *flag.String("basePath", "tmp/341554869", "file base path")
	thread 	  = *flag.Int("thread", 4, "download thread count")
)

func main() {
	flag.Parse()

	fmt.Printf("Spider Start! target:%s, deep: %d\n", targetUrl, deep)

	imageCh := make(chan *http.Image, 100)
	mainCh := make(chan int, thread)

	threadArr := make([]int, thread)
	for i := 0; i < thread; i ++ {
		threadArr[i] = i
		no := threadArr[i]
		go func() {
			fmt.Printf("D%d Start!\n", no)

			for {
				image, ok := <- imageCh
				if !ok {
					break
				}

				fmt.Printf("D%d  Download -> %s\n", no, image.FullName)

				func () {
					defer func () {
						if err := recover(); err != nil {
							fmt.Printf("Download failed %+v\n", err)
						}
					}()

					image.Download(basePath)
				}()
			}

			mainCh <- 0

			fmt.Printf("D%d end!\n", no)
		}()
	}

	pool := http.NewImagePool()

	res := http.Get(targetUrl)

	re := regexp.MustCompile(`https?:\/\/[\w\.\-\/]+\/[\w\-]+\.(?:jpg|jpeg|png|gif)`)
	urls := re.FindAllStringSubmatch(res, -1)
	fmt.Println("Image Urls Count -> ", len(urls))

	for i := 0; i < len(urls); i ++ {
		image, err := http.NewImage(urls[i][0])
		if err != nil {
			fmt.Printf("%s\n", err)
			continue
		}

		quality := pool.Exist(image.Uid)
		if quality >= image.GetQuality() {
			fmt.Printf("Exist higher quality image, skip %s %d - %d\n", image.FullName, quality, image.GetQuality())
			continue
		}

		pool.Push(image.Uid, image.GetQuality())
		imageCh <- image
	}

	close(imageCh)

	stopCount := 0
	for {
		<- mainCh
		stopCount ++

		if stopCount >= thread {
			break
		}
	}

	fmt.Println("Spider Exit ~ ")
}