package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sync"
)

const basePath = "tmp"

var (
	startAt = flag.Int("start", 0, "start page")
	max     = flag.Int("max", 10, "max page")
	thread  = flag.Int("thread", 2, "download thread count")
	host    = flag.String("host", "http://", "host")
)

var uniqMap map[string]bool

func flagHandler() error {
	flag.Parse()

	fmt.Printf("Spider Start! \nhost:%s\n, start:%d, max:%d \nthread:%d \n", *host, *startAt, *max, *thread)
	return nil
}

func main() {
	err := flagHandler()
	if err != nil {
		fmt.Println(err)
		return
	}

	uniqMap := make(map[string]bool)

	current := *startAt

	pCh := make(chan post, 10)
	iCh := make(chan Image, 100)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer close(pCh)
		defer wg.Done()

		fmt.Println("fetch posts thread start !")
		for {
			fmt.Printf("fetch posts -> page: %d\n", current)
			posts := getPosts(current)
			fmt.Printf("fetch posts -> post length: %d\n", len(posts))

			for _, p := range posts {
				pCh <- p
			}

			if current >= *max {
				break
			}

			current += 1
		}
		fmt.Println("fetch posts thread end !")
	}()

	wg.Add(1)
	go func() {
		fmt.Println("fetch images thread start !")
		defer close(iCh)
		defer wg.Done()

		for {
			p, ok := <-pCh
			if !ok {
				break
			}

			fmt.Printf("fetch images -> postId: %s\n", p.Id)

			images := getPostDetail(p)
			fmt.Printf("fetch images -> length: %d\n", len(images))

			current += 1

			for _, i := range images {
				if _, ok := uniqMap[i.Id]; !ok {
					uniqMap[i.Id] = true

					iCh <- i
				}
			}
		}
		fmt.Println("fetch images thread end !")
	}()

	threadArr := make([]int, *thread)

	for i := 0; i < *thread; i++ {
		threadArr[i] = i
		no := threadArr[i]
		wg.Add(1)
		go func() {
			fmt.Printf("[D%d] Start!\n", no)

			for {
				image, ok := <-iCh
				if !ok {
					break
				}

				fmt.Printf("[D%d]  Download -> %s\n", no, image.Id)

				func() {
					defer func() {
						if err := recover(); err != nil {
							fmt.Printf("[D%d] Download failed %+v\n", no, err)
						}
					}()

					image.Download(basePath)
				}()
			}

			fmt.Printf("[D%d] end!\n", no)
			wg.Done()
		}()
	}

	wg.Wait()

	fmt.Println("main thread exit !!!")
}

func getPostDetail(p post) []Image {
	res := Get(fmt.Sprintf("%s/?p=%s", *host, p.Id))
	reg := fmt.Sprintf(`%s/wp-content/uploads/user_files/1/bbs/images/([\d_]+).(jpg|png|jpeg)`, *host)
	re := regexp.MustCompile(reg)
	data := re.FindAllStringSubmatch(res, -1)
	fmt.Println("images -> lens ", len(data))

	var images []Image

	for _, item := range data {
		images = append(images, Image{Id: item[1], Url: item[0], Prefix: item[2], Post: p})
	}

	return images
}

func getPosts(page int) []post {
	var url string
	if page == 0 {
		url = *host
	} else {
		url = fmt.Sprintf("%s/wp-content/themes/LightSNS_1.6.35/module/more/data.php", *host)
	}
	res := Post(url, fmt.Sprintf(`type=all&page=%d&author_id=0`, page))

	fmt.Println("detail -> ", len(res))
	reg := fmt.Sprintf(`<h1 class="single bbs">\n<a href="%s/\?p=(\d+)" target="_blank">(.+)</a>`, *host)
	re := regexp.MustCompile(reg)
	data := re.FindAllStringSubmatch(res, -1)

	var posts []post

	for _, item := range data {
		posts = append(posts, post{Id: item[1], Name: item[2]})
	}

	return posts
}

type post struct {
	Id   string
	Name string
}

type Image struct {
	Id     string
	Url    string
	Prefix string
	Post   post
}

func (image Image) Download(basePath string) {
	os.MkdirAll(fmt.Sprintf("%s/%s-%s", basePath, image.Post.Id, image.Post.Name), os.ModePerm)

	filePath := fmt.Sprintf("%s/%s-%s/%s.%s", basePath, image.Post.Id, image.Post.Name, image.Id, image.Prefix)

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Create File Failed -> %+v\n", err)
		return
	}
	defer file.Close()

	Download(image.Url, file)

	file.Sync()
}
