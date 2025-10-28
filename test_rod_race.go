package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	l := launcher.New().Headless(true).Set("no-sandbox")
	defer l.Cleanup()

	controlURL, err := l.Launch()
	if err != nil {
		log.Fatal("Failed to launch:", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		log.Fatal("Failed to create page:", err)
	}
	defer page.Close()

	fmt.Println("Testing timeout with Context (3 seconds)...")
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	err = rod.Try(func() {
		page.Context(ctx).MustNavigate("https://www.dublab.com/archive/apientos-new-age-dance-show-10-23-25").MustWaitLoad()
	})
	
	if err != nil {
		fmt.Printf("Failed after %v: %v\n", time.Since(start), err)
	} else {
		fmt.Printf("Success after %v\n", time.Since(start))
	}
}
