package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/urfave/cli/v2"
)

type scriptInfo struct {
	Src     string `json:"src"`
	Content string `json:"content"`
}

var secretRegex = map[string]string{
	"Google API Key":                             `AIza[0-9A-Za-z-_]{35}`,
	"Google OAuth 2.0 Access Token":              `ya29.[0-9A-Za-z-_]+`,
	"GitHub Personal Access Token (Classic)":     `^ghp_[a-zA-Z0-9]{36}$`,
	"GitHub Personal Access Token (Fine-Grained": `^github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}$`,
	"GitHub OAuth 2.0 Access Token":              `^gho_[a-zA-Z0-9]{36}$`,
	"GitHub User-to-Server Access Token":         `^ghu_[a-zA-Z0-9]{36}$`,
	"GitHub Server-to-Server Access Token":       `^ghs_[a-zA-Z0-9]{36}$`,
	"GitHub Refresh Token":                       `^ghr_[a-zA-Z0-9]{36}$`,
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
	"Twilio":                                     `55[0-9a-fA-F]{32}`,
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
	"Generic Secret":                             `[s|S][e|E][c|C][r|R][e|E][t|T].*['|\"][0-9a-zA-Z]{32,45}['|\"]`,
	"Password in URL":                            `[a-zA-Z]{3,10}:\\/[^\\s:@]{3,20}:[^\\s:@]{3,20}@.{1,100}[\"'\s]`,
	"Slack Webhook URL":                          `https://hooks.slack.com/services/T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`,
}

// Return response from getting URL
func getContents(url string) (*string, string, error) {
	if url[:2] == "//" {
		url = "https:" + url
	} else if url == "" {
		return nil, url, fmt.Errorf("Attempted to get contents of empty URL")
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, url, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, url, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Read the entire text into a string
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, url, err
	}

	textString := string(bytes)
	if err != nil {
		return nil, url, err
	}

	return &textString, url, nil
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

// Return the script sources from the DOM
func getDOM(url string) ([]string, *string, error) {
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
	ctx, cancel = context.WithTimeout(taskCtx, 10*time.Second)
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
		result = append(result, currentString)
	}

	return result, nil
}

func getSecrets(text string, flags map[string]bool) map[string]string {
	//If the user enables the urls flag, we will search for URLs as well
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
	}

	//Search the provided text for any matches to the list of regex patterns
	var results = map[string]string{}
	for description, regex := range secretRegex {
		re := regexp.MustCompile(regex)
		matches := re.FindAllString(text, -1)
		for _, match := range matches {
			results[description] = match
		}
	}
	return results
}

func stringsCheck(url string, flags map[string]bool) ([]string, error) {
	//If the dom flag is set, we will search the DOM for strings
	if flags["dom"] {
		//Get the list of script sources from the DOM
		scripts, inline, err := getDOM(url)
		if err != nil {
			return nil, err
		}

		var strings []string
		if scripts != nil {
			//For each script source, go to the page and search for strings
			for _, script := range scripts {
				text, currUrl, err := getContents(script)
				if err != nil {
					return nil, err
				}
				s, err := getStrings(*text, flags)
				if err != nil {
					return nil, err
				}

				for _, str := range s {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + currUrl + ")"
					}
					strings = append(strings, str+location)
				}
			}
		}
		if inline != nil {
			//Search the inline scripts for any strings
			text, currUrl, err := getContents(*inline)
			if err != nil {
				return nil, err
			}
			s, err := getStrings(*text, flags)
			if err != nil {
				return nil, err
			}

			for _, str := range s {
				var location string
				if !flags["verify"] {
					location = ""
				} else {
					location = " (Location: " + currUrl + ")"
				}
				strings = append(strings, str+location)
			}
		}

		return strings, nil
		//If the dom flag is not set we will get the list of scripts from the HTML response
	} else {
		scripts, err := getScripts(url)
		if err != nil {
			return nil, err
		}

		var strings []string
		if scripts != nil {
			//For each script source, go to the page and search for strings
			for _, script := range scripts {
				text, currUrl, err := getContents(script)
				if err != nil {
					return nil, err
				}
				s, err := getStrings(*text, flags)
				if err != nil {
					return nil, err
				}

				for _, str := range s {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + currUrl + ")"
					}
					strings = append(strings, str+location)
				}
			}
		}

		return strings, nil
	}
}

func secretsCheck(url string, flags map[string]bool) ([]string, error) {
	//If the dom flag is set, we will search the DOM for secrets
	if flags["dom"] {
		//Get the list of script sources from the DOM
		scripts, inline, err := getDOM(url)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, url) //We will also check the original page, which means inline scripts will be checked as well

		var strings []string
		if scripts != nil {
			//For each script source, go to the page and search for secrets
			for _, script := range scripts {
				text, currUrl, err := getContents(script)
				if err != nil {
					return nil, err
				}
				s := getSecrets(*text, flags)
				for description, finding := range s {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + currUrl + ")"
					}
					strings = append(strings, "Possible "+description+" found: "+finding+location)
				}
			}
		}
		if inline != nil {
			//Search for secrets in inline scripts
			text, currUrl, err := getContents(*inline)
			if err != nil {
				return nil, err
			}
			s := getSecrets(*text, flags)
			for description, finding := range s {
				var location string
				if !flags["verify"] {
					location = ""
				} else {
					location = " (Location: " + currUrl + ")"
				}
				strings = append(strings, "Possible "+description+" found: "+finding+location)
			}
		}

		return strings, nil
	} else {
		//If the dom flag is not set we will get the list of scripts from the HTML response
		scripts, err := getScripts(url)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, url) //We will also check the original page, which means inline scripts will be checked as well

		var strings []string
		if scripts != nil {
			//For each script source, go to the page and search for secrets
			for _, script := range scripts {
				text, currUrl, err := getContents(script)
				if err != nil {
					return nil, err
				}
				s := getSecrets(*text, flags)
				for description, finding := range s {
					var location string
					if !flags["verify"] {
						location = ""
					} else {
						location = " (Location: " + currUrl + ")"
					}
					strings = append(strings, "Possible "+description+" found: "+finding+location)
				}
			}
		}
		return strings, nil
	}
}

// If secrets is true, then we are looking for secrets, otherwise we are looking for strings
func run(url string, flags map[string]bool) ([]string, error) {
	if flags["secrets"] {
		return secretsCheck(url, flags)
	} else {
		return stringsCheck(url, flags)
	}
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
					fmt.Print("\nSearching " + url + "\n")
					strings, err := run(url, flags)
					if err != nil {
						return err
					}

					if len(strings) == 0 {
						fmt.Println("No results found")
					}

					for _, str := range strings {
						fmt.Println(str)
					}
				}
			} else {
				url := cCtx.Args().First()

				if url == "" {
					return fmt.Errorf("no URL provided")
				}

				strings, err := run(url, flags)
				if err != nil {
					return err
				}

				if len(strings) == 0 {
					fmt.Println("No results found")
				}

				for _, str := range strings {
					fmt.Println(str)
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
