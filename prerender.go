package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/go-redis/redis/v7"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	taskCtx     context.Context
	redisClient *RedisClient
)

type cachePage struct {
	Url string
	Content string
}

func (cache *cachePage) GetRedisKey() string {
	return fmt.Sprintf("prerender:%x", md5.Sum([]byte(cache.Url)))
}

func (cache *cachePage) GetTtl() time.Duration {
	return 1 * time.Minute
}

func main() {
	redisClient = NewRedisClient()
	defer redisClient.Close()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	taskCtx = ctx
	defer ctxCancel()

	if err := chromedp.Run(taskCtx); err != nil {
		panic(err)
	}

	http.HandleFunc("/", ssrHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)

	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ssrHandler(w http.ResponseWriter, r *http.Request) {
	requestUrl, _ := url.ParseRequestURI(r.URL.RawQuery)
	if requestUrl == nil {
		return
	}

	cache := &cachePage{
		Url: requestUrl.String(),
	}

	err := redisClient.GetKey(cache.GetRedisKey(), cache)
	if err == redis.Nil {
		tabCtx, tabCancel := chromedp.NewContext(taskCtx)
		defer tabCancel()

		log.Print("Fetch url " + cache.Url)

		if err := chromedp.Run(tabCtx, GetHtmlContent(cache.Url, &cache.Content)); err != nil {
			log.Print(err)
		}

		err := redisClient.SetKey(cache.GetRedisKey(), cache, cache.GetTtl())
		if err != nil {
			log.Print(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprint(w, cache.Content)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetHtmlContent(url string, output *string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}

			*output, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			if err != nil {
				return err
			}

			return nil
		}),
	}
}
