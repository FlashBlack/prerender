package main

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"log"
	"net/http"
	"os"
)

var (
	taskCtx context.Context
)

func main() {
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
	if len(r.URL.Path) >= 6 && r.URL.Path[:6] == "/sdapi" {
		return
	}

	var buffer string
	var url = "https://ab.onliner.by" + r.RequestURI

	tabCtx, tabCancel := chromedp.NewContext(taskCtx)
	defer tabCancel()

	log.Print("Fetch url " + url)

	if err := chromedp.Run(tabCtx, getHtmlContent(url, &buffer)); err != nil {
		log.Print(err)
	}

	_, err := fmt.Fprint(w, buffer)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func getHtmlContent(url string, output *string) chromedp.Tasks {
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
