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
	// Create timeout context (5 seconds to force timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Launch browser
	l := launcher.New().Headless(true).Set("no-sandbox")
	defer l.Cleanup()

	controlURL, err := l.Context(ctx).Launch()
	if err != nil {
		log.Fatal("Failed to launch:", err)
	}

	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer browser.Close()

	fmt.Println("Browser connected")

	// Create page - page should inherit browser's context
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		log.Fatal("Failed to create page:", err)
	}
	defer page.Close()

	fmt.Println("Navigating to dublab (should timeout in 5s)...")
	start := time.Now()
	
	if err := page.Navigate("https://www.dublab.com/schedule/5konvvnguabf9nb8h4uj7grong_20251030T153300Z/out-therea-field-recording-program"); err != nil {
		fmt.Printf("Navigate failed after %v: %v\n", time.Since(start), err)
		return
	}

	fmt.Printf("Navigated in %v!\n", time.Since(start))
}
