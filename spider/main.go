package main

import (
	"fmt"
	"flag"
	"errors"
	"regexp"
	"winse.com/spider/http"
)

var (
	targetUrl = flag.String("target", "https://zhuanlan.zhihu.com/p/118104970", "spider target url")
	basePath  = flag.String("path", "", "file base path")
	thread 	  = flag.Int("thread", 4, "download thread count")
	uniq 	  	= flag.Bool("uniq", false, "ignore same image")
)

func flagHandler() error {
	flag.Parse()

	if len(*basePath) == 0 {
		pathRe := regexp.MustCompile(`https?:\/\/[\w+\.\-\/]+\/(\d+)`)
		data := pathRe.FindAllStringSubmatch(*targetUrl, -1)

		if len(data) != 1 {
			return errors.New("analysis target url failed")
		}

		*basePath = fmt.Sprintf("tmp/%s", data[0][len(data[0]) - 1])
	}

	fmt.Printf("Spider Start! \ntarget:%s \nbasePath:%s \nthread:%d \nuniq: %v\n\n", *targetUrl, *basePath, *thread, *uniq)
	return nil
}

func main() {
	err := flagHandler()
	if err != nil {
		fmt.Println(err)
		return
	}

	pool := http.NewImagePool()

	ch := make(chan string, 100)
	mainCh := make(chan int, *thread)

	threadArr := make([]int, *thread)

	for i := 0; i < *thread; i ++ {
		threadArr[i] = i
		no := threadArr[i]
		go func() {
			fmt.Printf("[D%d] Start!\n", no)

			for {
				imageUrl, ok := <- ch
				if !ok {
					break
				}

				image, err := http.NewImage(imageUrl)
				if err != nil {
					fmt.Printf("[D%d] -> %s\n", no, err)
					continue
				}

				quality := pool.Exist(image.Uid)
				if *uniq && quality >= image.GetQuality() {
					fmt.Printf("[D%d] Exist higher quality image, skip %s %d - %d\n", no, image.FullName, quality, image.GetQuality())
					continue
				}

				pool.Push(image.Uid, image.GetQuality())

				fmt.Printf("[D%d]  Download -> %s\n", no, image.FullName)

				func () {
					defer func () {
						if err := recover(); err != nil {
							fmt.Printf("[D%d] Download failed %+v\n", no, err)
						}
					}()

					image.Download(*basePath)
				}()
			}

			mainCh <- 0

			fmt.Printf("[D%d] end!\n", no)
		}()
	}

	res := http.Get(*targetUrl)

	re := regexp.MustCompile(`https?:\/\/[\w\.\-\/]+\/[\w\-]+\.(?:jpg|jpeg|png|gif)`)
	urls := re.FindAllStringSubmatch(res, -1)
	fmt.Println("Image Urls Count -> ", len(urls))

	for i := 0; i < len(urls); i ++ {
		ch <- urls[i][0]
	}

	close(ch)

	stopCount := 0
	for {
		<- mainCh
		stopCount ++

		if stopCount >= *thread {
			break
		}
	}

	fmt.Println("Spider Exit ~ ")
}