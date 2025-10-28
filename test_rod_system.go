package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	// Launch browser using system Chromium (like production)
	chromiumPath := "/Applications/Chromium.app/Contents/MacOS/Chromium"  // Mac path
	if _, err := launcher.NewBrowser().Exists(chromiumPath); err != nil {
		chromiumPath = "/usr/bin/chromium-browser"  // Try Alpine path
	}

	l := launcher.New().
		Bin(chromiumPath).
		Headless(true).
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		Set("disable-extensions").
		Set("disable-plugins")

	fmt.Println("Using Chromium at:", chromiumPath)

	controlURL, err := l.Launch()
	if err != nil {
		log.Fatal("Failed to launch:", err)
	}
	defer l.Cleanup()

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer browser.Close()

	fmt.Println("Browser connected")

	// Create page
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		log.Fatal("Failed to create page:", err)
	}
	defer page.Close()

	// Set timeout BEFORE operations
	page = page.Timeout(10 * time.Second)

	fmt.Println("Navigating to dublab...")
	start := time.Now()
	if err := page.Navigate("https://www.dublab.com/archive/greg-belson-w-dj-luke-howard-hmd-divine-chord-gospel-show-10-25-25"); err != nil {
		fmt.Printf("Failed to navigate after %v: %v\n", time.Since(start), err)
		return
	}

	fmt.Printf("Navigated in %v! Waiting for load...\n", time.Since(start))
	if err := page.WaitLoad(); err != nil {
		fmt.Printf("Failed to wait for load: %v\n", err)
		return
	}

	fmt.Println("Page loaded!")
}
