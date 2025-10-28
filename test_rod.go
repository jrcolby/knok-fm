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
	// Launch browser
	l := launcher.New().
		Headless(true).
		Set("no-sandbox").
		Set("disable-web-security")

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
	if err := page.Navigate("https://www.dublab.com/archive/greg-belson-w-dj-luke-howard-hmd-divine-chord-gospel-show-10-25-25"); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	fmt.Println("Navigated! Waiting for load...")
	if err := page.WaitLoad(); err != nil {
		log.Fatal("Failed to wait for load:", err)
	}

	fmt.Println("Page loaded!")
}
