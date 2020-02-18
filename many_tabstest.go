package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

func main2() {
	dir, err := ioutil.TempDir("", "chromedp-manytabs")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Using Temp: %s for chrome's user-data-dir", dir)
	defer os.RemoveAll(dir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		// chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	mainWindowCtx, mainCancel := chromedp.NewContext(allocCtx)
	defer mainCancel()

	if err := chromedp.Run(mainWindowCtx); err != nil {
		log.Fatal(err)
	}

	lister(mainWindowCtx)

	var wg sync.WaitGroup

	numWorkers := 3
	for w := 0; w < numWorkers; w++ {
		tabCtx, _ := chromedp.NewContext(mainWindowCtx)
		if err := chromedp.Run(tabCtx); err != nil {
			log.Fatal(err)
		}

		wg.Add(1)

		go worker(tabCtx, &wg, w, 10, 1000)
	}
	wg.Wait()

	log.Printf("Done waiting for %d workers", numWorkers)
}

func lister(ctx context.Context) {
	url := `https://via.placeholder.com/320x200/f00/fff?text=Main+Window`

	if err := chromedp.Run(ctx, showPage(url, `img`)); err != nil {
		log.Fatal(err)
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, w, n, maxDelay int) {
	for i := 0; i < n; i++ {
		if err := chromedp.Run(ctx, showPage(workerImageURL(w, i), `img`)); err != nil {
			log.Fatal(err)
		}

		delay := rand.Intn(maxDelay)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	wg.Done()
}

func workerImageURL(w, i int) string {
	return fmt.Sprintf("https://via.placeholder.com/320x200/00f/fff?text=Worker:%d+Image:%d", w, i)
}

func showPage(urlstr, sel string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.WaitVisible(sel, chromedp.ByQuery),
	}
}