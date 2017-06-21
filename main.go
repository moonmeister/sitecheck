package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"golang.org/x/net/html"
)

type httpRequest struct {
	URL string
}

type httpStatus struct {
	URL    string        `json:"url"`
	Status int           `json:"status"`
	Rtime  time.Duration `json:"rtime"`
}

//type HttpRequests []HttpRequest

func httpHead(site string) (httpstatus httpStatus, err error) {

	//if var is not null check site status
	if len(site) != 0 {

		//validate site var
		validURL := govalidator.IsURL(site)

		//error if invalid
		if validURL == false {
			return httpstatus, errors.New("Invalid URL")
		}

		//record time to measure response time
		start := time.Now()

		//HEAD request URL
		resp, err := http.Head(site)

		//Stop Response time timer
		Rtime := time.Since(start)

		//handle error from HEAD request
		if err != nil {
			return httpstatus, errors.New("Error requesting HEAD")
		}

		//unescape URL before writeout
		usite, err := url.QueryUnescape(site)
		if err != nil {
			return httpstatus, errors.New("Error requesting Unescaping url")
		}

		//assemble JSON response
		httpstatus = httpStatus{
			URL:    usite,
			Status: resp.StatusCode,
			Rtime:  Rtime,
		}

		return httpstatus, nil
		//else return no var
	}

	return httpstatus, errors.New("No Site Requested!\n")
}

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over all of the Token's attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as defined in
	// the function definition
	return
}

// Extract all http** links from a given webpage
func crawl(url string, ch chan string, chFinished chan bool) {
	fmt.Println("Attempting to Crawl: \"" + url + "\"")

	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		fmt.Println("ERROR: " + strconv.Itoa(resp.StatusCode))
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			// Extract the href value, if there is one
			ok, href := getHref(t)
			if !ok {
				continue
			}

			// Make sure the url begines in http**

			switch strings.Split(href, ":")[0] {
			case "mailto", "http", "https", "ftp":
				/*if href == url {
				  continue
				}*/
				ch <- href
			default:
				if strings.LastIndex(url, "/") != len(url)-1 && strings.Index(href, "/") != 0 {
					url = url + "/"
				} else if strings.LastIndex(url, "/") == len(url)-1 && strings.Index(href, "/") == 0 {
					url = strings.TrimSuffix(url, "/")
				}
				ch <- (url + href)
			}

			/*hasProto := strings.Index(href, "http") == 0
						if hasProto {
							ch <- href
						}else if len(href) > 1 && strings.Index(href, "mailto") != 0 {
			        //fmt.Println("url2: " + url2)
			        if strings.LastIndex(url, "/") != len(url)-1 && strings.Index(href, "/") != 0 {
			          url = url + "/"
			        }
			        ch <- (url + href)
			      }else{
			        ch <- href
			      }*/
		}
	}
}

func main() {
	foundUrls := make(map[string]bool)
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	for _, url := range seedUrls {
		go crawl(url, chUrls, chFinished)
	}

	// Subscribe to both channels
	for c := 0; c < len(seedUrls); {
		select {
		case url := <-chUrls:
			foundUrls[url] = true
			//crawl(url, chUrls, chFinished)
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...

	fmt.Println("\nFound", len(foundUrls), "unique urls:")

	for url := range foundUrls {
		result, err := httpHead(url)
		if err != nil {
			print(err)
		}
		fmt.Printf("URL: %v STATUS: %v\n", result.URL, result.Status)
	}

	close(chUrls)
}
