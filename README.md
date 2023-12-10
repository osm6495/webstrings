<!-- PROJECT SHIELDS -->
<!--
*** I'm using markdown "reference style" links for readability.
*** Reference links are enclosed in brackets [ ] instead of parentheses ( ).
*** See the bottom of this document for the declaration of the reference variables
*** for contributors-url, forks-url, etc. This is an optional, concise syntax you may use.
*** https://www.markdownguide.org/basic-syntax/#reference-style-links
-->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Downloads][downloads-shield]][downloads-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]
[![Medium][medium-shield]][medium-url]



<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/osm6495/webstrings">
    <img src="resources/Logo.png" alt="Logo" width="300" height="300">
  </a>

  <h3 align="center">WebStrings</h3>

  <p align="center">
    An awesome README template to jumpstart your projects!
    <br />
    <a href="https://github.com/osm6495/webstrings"><strong>Report Bug or Request Feature»</strong></a>
  </p>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

The [strings](https://linux.die.net/man/1/strings) command is incredibly useful for reverse engineering and pentesting local programs, but there hasn't a simple way to do the same on the web. That's where webstrings comes in. With a simple `webstrings "https://example.com"`, you can return all of the strings found in the client-side code of the website.

One of the main use cases for webstrings is looking for exposed secrets, which is why you can also use the `-s` option to search for common secret patterns such as API keys or the `-su` option to search for any URLs as well.

For cases where things get complicated and the code is generated dynamically on page load, you can even use the `-d` option to load the page in a headless browser and get all of the script sources from the DOM.

The opportunities for using this are endless. Maybe try running this daily on a website you are interested in and piping the output to a file. You could compare that to the next day and alert yourself when anything new shows up. Maybe use the output as a wordlist when you are trying to access a pentest target's API. If you make anything cool with this, I'd love to hear about it.


<!-- GETTING STARTED -->
## Getting Started

Here is how to set up webstrings:

### Prerequisites

You will need google chrome installed if you plan on using the DOM searching flag
* Linux - Apt
  ```sh
  sudo apt install google-chrome-stable
  ```
If you are on Windows you may need to run this in a Golang docker container to get the DOM flag to work, as the chromedp Golang library that is used as a headless browser is looking for the google-chrome executable, rather than chrome.exe.

### Installation
The easiest way to use webstrings is to install the latest release
1. Download the latest release: https://github.com/osm6495/webstrings/releases
2. You can add the binary to your bin or path to make it easier to run, but otherwise you can get started with:
    ```sh
    ./webstrings -h
    ```
----
You can also install with Golang:
1. Make sure that `$GOPATH/bin` is in your `$PATH`, because that's where this gets installed:
    ```
    go install github.com/osm6495/webstrings@latest
    ```
----
If you would like to build from the source code instead, that process is easy as well:
1. You'll need Golang installed. If you don't already have it you can get it from the [website](https://go.dev/dl/) or use your package manager, like apt on linux:
    ```sh
    sudo apt install golang
    ```
2. Clone down the repository and navigate into it:
    ```sh
    git clone https://github.com/osm6495/webstrings
    cd webstrings
    ```
3. 




<!-- USAGE EXAMPLES -->
## Usage

<div align="center">
  <a href="https://github.com/osm6495/webstrings">
    <img src="resources/Manual.png" alt="Logo" width="800" height="320">
  </a>
</div>

Using webstrings is as simple as:
```sh
webstrings "http://example.com"
```
(Although there isn't any code on that URL, so you won't get any findings.)
<br>By default, webstrings is searching for strings. It will go to the URL, get any scripts mentioned in the page's response, and check those and the original response for any strings.

You can use the `-d` flag to use a headless browser to search the DOM instead. This will allow you to also catch and scripts that are loaded in dynamically as the page loads.

By default, the `-v` flag is disabled so that the output is more minimal, but when you find a string and you want to know where to find it on the site you can run the CLI again with that flag and it will include the URL where it found the string. Then you can go to that URL, which is usually a link to a script, and search for the string.

The `-s` flag can be used to search for secrets, rather than just strings. This will use regex patterns to search the site and scripts for any API keys or other sensitive information that may be exposed.

Many of these regex patterns are too generalized and will produce a lot of false positives, so those are now behind the `-n` flag. By default you should only get a response if it matches the specific format of a secret, but if you want anything that possibly fits the shape of a secret you can use the `-n` flag to open the floodgates and mention anything noteworthy.

The `-u` flag can be used to search the site and scripts for any URLs. By default it will only look for urls that start with `http://` or `https://`, but if you combine the `-u` and `-n` flags, you will use a more general regex for URLs which would include URLs like `example.com`

If you want to check a list of sites, you can use the `-f` flag to input the path to a list file of URLs, rather than a single URL.

Importantly, these flags can all be combined so feel free to experiment with things like:
```sh
webstrings -funds linkfile.txt
```
which would go through each URL in `linkfile.txt` and search the dom for any secrets, including URLs and all of the rules that generate large amounts of false positives.

If you want to output to a file you can pipe the output of the command to a file in Linux:
```sh
webstrings "https://example.com" > out.txt
```

### Validating your Findings
To find where your secret finding is in the webpage:
1. Use the `-v` flag to have the URL of the finding output with your findings.
2. Make an HTTP GET request to the URL in the `Location:` in the output of the finding.
3. Search the response body with the regex string from the list of Secrets Regex Patterns that matches your finding 
    - If you don't have a tool for this, you can paste the response into a file and open it with VSCode. The `Ctrl+F` search has a button with `Alt+R` that allows you to use regex patterns. You could also paste it into an online regex testing site like regex101.com or regexr.com


### Secrets Regex Strings
```
    Secrets flag:
        "Google API Key":                             `AIza[0-9A-Za-z-_]{35}`,
        "Google OAuth 2.0 Refresh Token":             `1/[0-9A-Za-z-]{43}|1/[0-9A-Za-z-]{64}`,
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

    Noisy flag:
		"Google OAuth 2.0 Auth Code":                 `4/[0-9A-Za-z-_]+`,
		"Google Cloud Platform API Key":              `[A-Za-z0-9_]{21}--[A-Za-z0-9_]{8}`,
		"Heroku API Key":                             `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
    
    Url flag:
        "URL":                                        `(http(s)?:\/\/.)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`,
```



<!-- ROADMAP -->
## Roadmap

- [ ] File Output Formats
    - This can currently be done by piping the output of the CLI to a file:<br>`webstrings "https://example.com" > out.txt`
    - Would be nice to have JSON and other formats, as well as options for keeping progress output in the console and only outputting findings to the file
- [] Better minified JS parsing
    - Minified JS often ends up outputting huge chunks of JS code rather than actual strings, quotes and other delimiters are sometimes inaccurate. Would need to find another way to parse this, maybe some kind of AST.
- [ ] Allow regex input files to add secret patterns

See the [open issues](https://github.com/osm6495/webstorm/issues) for a full list of proposed features (and known issues).




<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request




<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE.txt` for more information.




<!-- CONTACT -->
## Contact

Owen McCarthy - contact@owen.biz


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/osm6495/webstrings
[contributors-url]: https://github.com/osm6495/webstrings/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/osm6495/webstrings
[forks-url]: https://github.com/osm6495/webstrings/network/members
[stars-shield]: https://img.shields.io/github/stars/osm6495/webstrings
[stars-url]: https://github.com/osm6495/webstrings/stargazers
[issues-shield]: https://img.shields.io/github/issues/osm6495/webstrings
[issues-url]: https://github.com/osm6495/webstrings/issues
[license-shield]: https://img.shields.io/github/license/osm6495/webstrings
[license-url]: https://github.com/osm6495/webstrings/LICENSE
[downloads-shield]: https://img.shields.io/github/downloads/osm6495/webstrings/total
[downloads-url]: https://github.com/osm6495/webstrings/graphs/contributors
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-blue
[linkedin-url]: https://www.linkedin.com/in/owen-mccarthy-060827192/
[medium-shield]: https://img.shields.io/badge/-Medium-white
[medium-url]: https://medium.com/@omccarth333
[product-screenshot]: images/screenshot.png