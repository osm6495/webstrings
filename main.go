package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	netUrl "net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/sourcegraph/conc/pool"
	"github.com/urfave/cli/v2"
	"golang.org/x/time/rate"
)

type scriptInfo struct {
	Src     string `json:"src"`
	Content string `json:"content"`
}

type URLQueue struct {
	mu    sync.Mutex
	queue []string
}

func (q *URLQueue) Push(url string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = append(q.queue, url)
}

func (q *URLQueue) Pop() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.queue) == 0 {
		return ""
	}
	url := q.queue[0]
	q.queue = q.queue[1:]
	return url
}

var outputMutex = sync.Mutex{}

var secretRegex = map[string]string{
	"Google API Key":                             `AIza[0-9A-Za-z-_]{35}`,
	"Google OAuth 2.0 Access Token":              `ya29.[0-9A-Za-z-_]+`,
	"GitHub Personal Access Token (Classic)":     `ghp_[a-zA-Z0-9]{36}`,
	"GitHub Personal Access Token (Fine-Grained": `github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}`,
	"GitHub OAuth 2.0 Access Token":              `gho_[a-zA-Z0-9]{36}`,
	"GitHub User-to-Server Access Token":         `ghu_[a-zA-Z0-9]{36}`,
	"GitHub Server-to-Server Access Token":       `ghs_[a-zA-Z0-9]{36}`,
	"GitHub Refresh Token":                       `ghr_[a-zA-Z0-9]{36}`,
	"Foursquare Secret Key":                      `R_[0-9a-f]{32}`,
	"Picatic API Key":                            `sk_live_[0-9a-z]{32}`,
	"Stripe Standard API Key":                    `sk_live_[0-9a-zA-Z]{24}`,
	"Stripe Restricted API Key":                  `sk_live_[0-9a-zA-Z]{24}`,
	"Square Access Token":                        `sqOatp-[0-9A-Za-z-_]{22}`,
	"Square OAuth Secret":                        `q0csp-[ 0-9A-Za-z-_]{43}`,
	"Paypal / Braintree Access Token":            `access_token,production$[0-9a-z]{161[0-9a,]{32}`,
	"Amazon Marketing Services Auth Token":       `amzn.mws.[0-9a-f]{8}-[0-9a-f]{4}-10-9a-f1{4}-[0-9a,]{4}-[0-9a-f]{12}`,
	"Mailgun API Key":                            `key-[0-9a-zA-Z]{32}`,
	"MailChimp":                                  `[0-9a-f]{32}-us[0-9]{1,2}`,
	"Slack OAuth v2 Bot Access Token":            `xoxb-[0-9]{11}-[0-9]{11}-[0-9a-zA-Z]{24}`,
	"Slack OAuth v2 User Access Token":           `xoxp-[0-9]{11}-[0-9]{11}-[0-9a-zA-Z]{24}`,
	"Slack OAuth v2 Configuration Token":         `xoxe.xoxp-1-[0-9a-zA-Z]{166}`,
	"Slack OAuth v2 Refresh Token":               `xoxe-1-[0-9a-zA-Z]{147}`,
	"Slack Webhook":                              `T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`,
	"AWS Access Key ID":                          `AKIA[0-9A-Z]{16}`,
	"Google Cloud Platform OAuth 2.0":            `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	"Heroku OAuth 2.0":                           `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	"Facebook Access Token":                      `EAACEdEose0cBA[0-9A-Za-z]+`,
	"Facebook OAuth":                             `[f|F][a|A][c|C][e|E][b|B][o|O][o|O][k|K].*['|\"][0-9a-f]{32}['|\"]`,
	"Twitter Username":                           `/(^|[^@\w])@(\w{1,15})\b/`,
	"Twitter Access Token":                       `[1-9][0-9]+-[0-9a-zA-Z]{40}`,
	"Cloudinary URL":                             `cloudinary://.*`,
	"Firebase URL":                               `.*firebaseio\.com`,
	"RSA Private Key":                            `-----BEGIN RSA PRIVATE KEY-----`,
	"DSA Private Key":                            `-----BEGIN DSA PRIVATE KEY-----`,
	"EC Private Key":                             `-----BEGIN EC PRIVATE KEY-----`,
	"PGP Private Key":                            `-----BEGIN PGP PRIVATE KEY BLOCK-----`,
	"Generic API Key":                            `[a|A][p|P][i|I][_]?[k|K][e|E][y|Y].*['|\"][0-9a-zA-Z]{32,45}['|\"]`,
	"Password in URL":                            `[a-zA-Z]{3,10}:\\/[^\\s:@]{3,20}:[^\\s:@]{3,20}@.{1,100}[\"'\s]`,
	"Slack Webhook URL":                          `https://hooks.slack.com/services/T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`,
}

// getContents connects to the URL and gets the page contents
//
// Parameters:
//   - ctx: The context for the search, used to cancel the search if needed and to create the HTTP request.
//   - url: The URL to search.
//   - baseUrl: The base URL to use if the URL is a relative URL.
//
// Returns:
//   - *string: A pointer to a string containing the page content.
//   - error
func getContents(ctx context.Context, url string, baseUrl string) (*string, error) {
	if url == "" {
		return nil, fmt.Errorf("Attempted to get contents of empty URL")
		//Check if the URL is a relative URL, if so, append the base URL
	} else if url[:1] == "/" {
		url = baseUrl + url
	}

	//Needs to come after the if statement above to allow relative URLS, otherwise they will get prefixed with https://
	parsedUrl, err := netUrl.Parse(url)
	if err != nil {
		return nil, err
	}

	if parsedUrl.Scheme == "" {
		baseUrl = "https://" + url
		url = "https://" + url
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("Warning - Attempted HTTP GET request creation of %s failed: %s", url, err)
		return nil, nil
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Warning - Attempted HTTP GET of %s failed: %s", url, err)
		return nil, nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		//Non-breaking error
		fmt.Printf("Warning - Attempted HTTP GET of %s returned status code error: %s\n", url, res.Status)
		return nil, nil
	}

	// Read the entire text into a string
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	textString := string(bytes)
	if err != nil {
		return nil, err
	}

	return &textString, nil
}

// getScripts get the list of script source links from the HTML of the input text
//
// Parameters:
//   - textString: A pointer to a string containing the page content to search.
//
// Returns:
//   - []string: A slice of strings containing the script source links.
//   - error
func getScripts(textString *string) ([]string, error) {
	body := strings.NewReader(*textString)

	//goquery is used to search for script tags with src attributes
	doc, err := goquery.NewDocumentFromReader(body)
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

// getDom opens a headless browser and navigates to the provided URL, then gets the script source links and inline scripts from the DOM
//
// This uses chromedp to get the script source links, but if it is possible to get the page contents with the same request that gets the DOM it is possible to reduce
// the number of requests needed, since currently getContents is still required in the search function when searching for secrets
//
// Parameters:
//   - parentCtx: The context for the search, used to cancel the search if needed and to pass to the chromedp context
//   - url: The URL to search.
//
// Returns:
//   - []string: A slice of strings containing the script source links.
//   - *string: A pointer to a string containing the inline script.
//   - error
func getDOM(parentCtx context.Context, url string) ([]string, *string, error) {
	// Create a chromedp context
	ctx, cancel := chromedp.NewContext(parentCtx)
	defer cancel()

	// Navigate to the page and get the list of script information (src and content)
	var scripts []scriptInfo
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
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
	var inline string
	// Process and print the script information
	for _, script := range scripts {
		if script.Src != "" {
			links = append(links, script.Src)
		} else if script.Content != "" {
			inline = script.Content
		}
	}

	if len(links) == 0 && inline != "" {
		return nil, nil, fmt.Errorf("no scripts found")
	} else if len(links) == 0 {
		return nil, &inline, nil
	} else {
		return links, nil, nil
	}
}

// getStrings is the function that takes in the content from a URL response or inline script and searches for strings
//
// Parameters:
//   - text: The text to search for strings.
//   - flags: The flags that the user input when using the CLI.
//
// Returns:
//   - []string: A slice of strings containing the findings.
func getStrings(text string, flags map[string]bool) ([]string, error) {
	inString := false
	currentString := ""
	escaped := false

	var result []string
	for _, char := range text {
		switch {
		case char == '"' || char == '\'' || char == '`':
			if inString {
				if escaped {
					// This is an escaped delimiter, add it to the current string
					currentString += "\\" + string(char)
					escaped = false
				} else {
					// End of the string, add to the channel
					if currentString != "" {
						result = append(result, currentString)
					}
					currentString = ""
					inString = false
				}
			} else {
				// Start of a new string
				inString = true
			}
		case char == '\\':
			if inString {
				// This is a backslash, mark the next character as escaped
				escaped = true
			}
		case inString:
			// Inside a string, add the character to the current string
			if char != '"' && char != '\'' && char != '`' {
				currentString += string(char)
			}
			escaped = false
		}
	}

	// Check for multiline strings using backticks (`) as delimiters
	if inString && strings.HasSuffix(currentString, "`") {
		result = append(result, currentString)
		currentString = ""
		inString = false
	}

	if inString {
		if flags["noisy"] {
			result = append(result, currentString)
		} else {
			//Compile the regex patterns to check for unwanted minified js code
			functionPattern := regexp.MustCompile(`function\(`)
			varPattern := regexp.MustCompile(`\bvar\b`)
			returnPattern := regexp.MustCompile(`\breturn\b`)

			functionMatch := functionPattern.MatchString(currentString)
			varMatch := varPattern.MatchString(currentString)
			returnMatch := returnPattern.MatchString(currentString)

			//Only add the string if it does not contain minified js code
			if !(functionMatch && varMatch && returnMatch) {
				result = append(result, currentString)
			}
		}

		result = append(result, currentString)
	}

	return result, nil
}

// getSecrets is the function that takes in the content from a URL response or inline script and searches for secrets using regex patterns
//
// Parameters:
//   - text: The text to search for secrets.
//   - flags: The flags that the user input when using the CLI.
//
// Returns:
//   - map[string][]string: A map of the secret description to a slice of strings containing the findings.
//     Example: {"URL": ["https://example.com", "https://example2.com"], "GitHub Personal Access Token (Classic)": ["ghp_123456789023456789012345678902345678"]}
func getSecrets(text string, flags map[string]bool) map[string][]string {
	//If the user enables the urls flag, we will append a URL regex to the global regex map
	if flags["urls"] && flags["noisy"] {
		//Use the noisy URL regex pattern (Does not require http(s)://)
		secretRegex["URL"] = `(http(s)?:\/\/.)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`
	} else if flags["urls"] {
		//If only using the urls flag, use the default URL regex pattern (Requires http(s)://)
		secretRegex["URL"] = `https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	}
	if flags["noisy"] {
		secretRegex["Google OAuth 2.0 Auth Code"] = `4/[0-9A-Za-z-_]+`
		secretRegex["Google Cloud Platform API Key"] = `[A-Za-z0-9_]{21}--[A-Za-z0-9_]{8}`
		secretRegex["Heroku API Key"] = `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`
		secretRegex["Google OAuth 2.0 Refresh Token"] = `1/[0-9A-Za-z-]{43}|1/[0-9A-Za-z-]{64}`
		secretRegex["Generic Secret"] = `[s|S][e|E][c|C][r|R][e|E][t|T].*['|\"][0-9a-zA-Z]{32,45}['|\"]`
		secretRegex["Twilio"] = `55[0-9a-fA-F]{32}`
	}

	//Compile the regex patterns to check for unwanted minified js code
	functionPattern := regexp.MustCompile(`function\(`)
	varPattern := regexp.MustCompile(`\bvar\b`)
	returnPattern := regexp.MustCompile(`\breturn\b`)

	//Search the provided text for any matches to the list of regex patterns
	var results = map[string][]string{}
	for description, regex := range secretRegex {
		re := regexp.MustCompile(regex)
		matches := re.FindAllString(text, -1)
		for _, match := range matches {
			if flags["noisy"] {
				results[description] = append(results[description], match)
			} else {
				functionMatch := functionPattern.MatchString(match)
				varMatch := varPattern.MatchString(match)
				returnMatch := returnPattern.MatchString(match)

				//Only add the match if it does not contain minified js code
				if !(functionMatch && varMatch && returnMatch) {
					results[description] = append(results[description], match)
				}
			}
		}
	}
	return results
}

// The search function searches a URL for strings or secrets
//
// This is the function that handles the logic for doing different searches for strings or secrets based
// on the flags provided by the user. It also handles the logic for searching the DOM if the user enables
// the dom flag.
//
// Parameters:
//   - ctx: The context for the search, used to cancel the search if needed and to pass to other functions.
//   - url: The URL to search.
//   - flags: The flags that the user input when using the CLI.
//   - urlQueue: A pointer to the URLQueue with the input URLs or any found during the search.
//
// Returns:
//   - []string: A slice of strings containing the results of the search.
//   - error
func search(ctx context.Context, url string, flags map[string]bool, urlQueue *URLQueue) ([]string, error) {
	var out []string
	if url == "" {
		return nil, fmt.Errorf("Attempted to search empty URL")
	}

	textString, err := getContents(ctx, url, url)
	if err != nil {
		return nil, err
	}

	var inline *string
	var scripts []string
	if flags["dom"] {
		//Currently getDOM can ONLY be used to get script sources, so both getContents and getDOM must be used
		scripts, inline, err = getDOM(ctx, url)
		if err != nil {
			return nil, err
		}

		if scripts != nil {
			for _, script := range scripts {
				//Check if the script is a relative URL, if so, append the base URL
				if script[:1] == "/" {
					script = url + script
				}
				urlQueue.Push(script)
			}
		}
	} else {
		scripts, err := getScripts(textString)
		if err != nil {
			return nil, err
		}
		if scripts != nil {
			for _, script := range scripts {
				//Check if the script is a relative URL, if so, append the base URL
				if script[:1] == "/" {
					script = url + script
				}
				urlQueue.Push(script)
			}
		}
	}

	if flags["secrets"] {
		var s map[string][]string
		//getContent can return a nil pointer if the request fails
		if textString != nil {
			s = getSecrets(*textString, flags)
		}

		//Append inline findings to the output as well
		if inline != nil {
			s2 := getSecrets(*inline, flags)
			if s2 != nil {
				for description, findings := range s2 {
					if existingValues, ok := s[description]; ok {
						s[description] = append(existingValues, findings...)
					} else {
						s[description] = findings
					}
				}
			}
		}

		if s != nil {
			for description, findings := range s {
				for _, finding := range findings {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + url + ")"
					}
					out = append(out, "Possible "+description+" found: "+finding+location)
				}
			}
		}
	} else {
		var s []string
		//getContent can return a nil pointer if the request fails
		if textString != nil {
			s, err = getStrings(*textString, flags)
		}
		if err != nil {
			return nil, err
		}

		//Append inline findings to the output as well
		if inline != nil {
			s2, err := getStrings(*inline, flags)
			if err != nil {
				return nil, err
			}
			if s2 != nil {
				for _, str := range s2 {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + url + ")"
					}
					out = append(out, str+location)
				}
			}
		}

		if s != nil {
			for _, str := range s {
				var location string
				if !flags["verify"] {
					location = ""
				} else {
					location = " (Location: " + url + ")"
				}
				out = append(out, str+location)
			}
		}
	}

	searchingMsg := fmt.Sprintf("\nSearching %s...\n", url)

	//Lock the outputMutex to prevent multiple goroutines from printing at the same time (searching1, result1, searching2, result2, etc.)
	outputMutex.Lock()
	defer outputMutex.Unlock()
	fmt.Print(searchingMsg)
	if out != nil {
		fmt.Println(out)
	} else {
		fmt.Println("No results found")
	}
	return out, nil
}

// The run function creates goroutines to search the provided URLS for strings or secrets
//
// Parameters:
//   - urlQueue: A pointer to the URLQueue with the input URLs or any found during the search.
//   - flags: The flags that the user input when using the CLI.
//
// Returns:
//   - error
//   - Output is printed to stdout in the search function, so no return value is needed.
func run(urlQueue *URLQueue, flags map[string]bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Limit the number of concurrent requests to 1 per second
	limiter := rate.NewLimiter(1, 1)

	pool := pool.NewWithResults[[]string]().WithContext(ctx)
	for _, url := range urlQueue.queue {
		err := limiter.Wait(ctx)
		if err != nil {
			return err
		}
		url := url //Capture the loop variable to make sure it isn't shared between goroutines
		pool.Go(func(ctx context.Context) ([]string, error) {
			return search(ctx, url, flags, urlQueue)
		})
	}

	_, err := pool.Wait()
	if err != nil {
		return err
	}

	//Output is printed in the search function, in order to output as each goroutine completes rather than after all are finished
	return nil
}

func main() {
	cli.AppHelpTemplate = `NAME:
	{{.Name}} - {{.Usage}}
 USAGE:
	{{.HelpName}} {{if .VisibleFlags}}{options}{{end}} [URL]
	{{if len .Authors}}
 AUTHOR:
	{{range .Authors}}{{ . }}{{end}}
	{{end}}{{if .Commands}}
 COMMANDS:
 {{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
 OPTIONS:
	{{range .VisibleFlags}}{{.}}
	{{end}}{{end}}{{if .Copyright }}
 COPYRIGHT:
	{{.Copyright}}
	{{end}}{{if .Version}}
 VERSION:
	{{.Version}}
	{{end}}
 `
	app := &cli.App{
		Name:  "webstrings",
		Usage: "Search web responses for strings or secrets",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dom",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "search the DOM for strings or secrets using a headless browser",
			},
			&cli.BoolFlag{
				Name:    "secrets",
				Aliases: []string{"s"},
				Value:   false,
				Usage:   "enable secrets search mode",
			},
			&cli.BoolFlag{
				Name:    "urls",
				Aliases: []string{"u"},
				Value:   false,
				Usage:   "includes any possible URLS as secret findings",
			},
			&cli.BoolFlag{
				Name:    "noisy",
				Aliases: []string{"n"},
				Value:   false,
				Usage:   "include secret regex patterns that produce a lot of false positives",
			},
			&cli.BoolFlag{
				Name:    "verify",
				Aliases: []string{"v"},
				Value:   false,
				Usage:   "include locations for findings",
			},
			&cli.BoolFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Value:   false,
				Usage:   "use a file as input instead of a single URL, format should be URLs separated by newlines",
			},
		},
		UseShortOptionHandling: true, //Allows -sd or -ds to be used instead of -s -d
		Action: func(cCtx *cli.Context) error {
			//Get a map of all the flags and their values
			flags := map[string]bool{}
			for _, flag := range cCtx.FlagNames() {
				flags[flag] = cCtx.Bool(flag)
			}

			if !flags["secrets"] && flags["urls"] {
				fmt.Println("URLS flag is only available in secrets mode, continuing with only strings")
			}

			urlQueue := &URLQueue{}
			if flags["file"] {
				path := cCtx.Args().First()

				if path == "" {
					return fmt.Errorf("no file path provided")
				}

				file, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				for _, url := range strings.Split(string(file), "\n") {
					urlQueue.Push(url)
				}

				err = run(urlQueue, flags)
				if err != nil {
					return err
				}
			} else {
				url := cCtx.Args().First()

				if url == "" {
					return fmt.Errorf("no URL provided")
				}

				parsedUrl, err := netUrl.Parse(url)
				if err != nil {
					return err
				}

				if parsedUrl.Scheme == "" {
					url = "https://" + url
				}

				urlQueue.Push(url)
				err = run(urlQueue, flags)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
