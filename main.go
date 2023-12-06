package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type scriptInfo struct {
	Src     string `json:"src"`
	Content string `json:"content"`
}

// Gets the list of script src links from the HTML response of the original URL
func getScripts(url string) ([]string, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var scripts []string
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		scriptSrc, exists := s.Attr("src")
		if exists {
			scripts = append(scripts, scriptSrc)
		}
	})

	return scripts, nil
}

func getDOM(url string) ([]string, io.ReadCloser, error) {
	// Create a chromedp context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set Chrome options for headless execution
	options := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
	}

	// Create an allocator with the given options
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, options...)
	defer allocCancel()

	// Use the allocator context to create a new chromedp context
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// Run a series of tasks with a timeout to ensure proper shutdown
	ctx, cancel = context.WithTimeout(taskCtx, 30*time.Second)
	defer cancel()

	// Navigate to the page and get the list of script information (src and content)
	var scripts []scriptInfo
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://www.example.com`),
		chromedp.WaitVisible(`body`, chromedp.ByQuery), // Wait for the body to be visible to ensure the page is loaded
		chromedp.Evaluate(`
			[...document.scripts].map(script => ({
				src: script.src,
				content: script.src ? '' : script.textContent,
			}))`, &scripts),
	)
	if err != nil {
		return nil, nil, err
	}

	var links []string
	var inline io.ReadCloser
	// Process and print the script information
	for _, script := range scripts {
		if script.Src != "" {
			links = append(links, script.Src)
		} else if script.Content != "" {
			inline = io.NopCloser(strings.NewReader(script.Content))
		}
	}

	if len(links) == 0 && inline != nil {
		return nil, nil, fmt.Errorf("no scripts found")
	} else if len(links) == 0 {
		return nil, inline, nil
	} else {
		return links, nil, nil
	}
}

// Get the JS code from the script src link
func getCode(url string, resultChan chan<- string) error {

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	scanFile(res.Body, resultChan)
	return nil
}

// Get all of the strings in a provided file
func scanFile(text io.ReadCloser, resultChan chan<- string) {
	stringChan := make(chan string, 100)
	var innerWG sync.WaitGroup
	for _, delimeter := range []rune{'\'', '"', '`'} {
		innerWG.Add(1)
		go getStrings(text, stringChan, &innerWG, delimeter)
	}

	// Close the result channel once all goroutines are done
	go func() {
		innerWG.Wait()
		close(stringChan)
	}()

	for str := range stringChan {
		resultChan <- str
	}
}

// Get the strings in a provided file that are closed with the provided string delimiter (', ", or `)
func getStrings(text io.ReadCloser, stringChan chan<- string, wg *sync.WaitGroup, delimiter rune) error {
	defer wg.Done()

	scanner := bufio.NewScanner(text)
	inString := false
	currentString := ""

	for scanner.Scan() {
		line := scanner.Text()

		for _, char := range line {
			switch {
			case char == delimiter:
				if inString {
					// End of the string, add to the channel
					stringChan <- currentString
					currentString = ""
					inString = false
				} else {
					// Start of a new string
					inString = true
					currentString += string(char)
				}
			case inString:
				// Inside a string, add the character to the current string
				currentString += string(char)
			}
		}

		// Check for multiline strings using backticks (`) as delimiters
		if inString && delimiter == '`' && strings.HasSuffix(currentString, "`") {
			stringChan <- currentString
			currentString = ""
			inString = false
		}
	}

	if inString {
		stringChan <- currentString
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	url := "https://google.com"

	_, _ = getScripts(url)

	scripts, inline, err := getDOM(url)
	if err != nil {
		panic(err)
	}

	if scripts != nil {
		resultChan := make(chan string, 100)
		var wg sync.WaitGroup

		for _, script := range scripts {
			wg.Add(1)
			go getCode(script, resultChan)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for str := range resultChan {
			fmt.Println(str)
		}
	}

	if inline != nil {
		inlineChan := make(chan string, 100)
		scanFile(inline, inlineChan)
		close(inlineChan)
		for str := range inlineChan {
			fmt.Println(str)
		}
	}

}
