package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	os.Exit(rMain())
}

func rMain() int {
	var storeUrl = flag.String("u", "", "url to the category you want to find all the deals on")

	flag.Parse()

	var err error
	var baseUrl *url.URL
	var page *html.Node
	var nodes []*html.Node
	var resp []byte

	//parse url for future shenanigans
	baseUrl, err = url.Parse(*storeUrl)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error parsing url:", err)
		return 1
	}
	baseUrl.RawQuery = ""

	for {
		resp, err = getPageWithUA(*storeUrl)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error downloading page:", err)
			return 1
		}

		page, err = html.Parse(bytes.NewReader(resp))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error parsing page:", err)
			return 1
		}

		//get listings
		nodes = htmlquery.Find(page, "//span[contains(text(), \"savings are available\")]/../h3/a")
		for _, e := range nodes {
			fmt.Print(strings.ReplaceAll(e.FirstChild.Data, "&nbsp;", " ") + " | ")
			for _, a := range e.Attr {
				if a.Key == "href" {
					fmt.Println(baseUrl.Scheme + "://" + baseUrl.Host + a.Val)
					break
				}
			}
		}

		//check if next button exists; go to next page if yes
		nodes = htmlquery.Find(page, "//a[span[text()=\"Next\"]]/@data-page-number")
		if len(nodes) == 0 {
			return 0
		}
		*storeUrl = baseUrl.String() + "?page=" + nodes[0].FirstChild.Data
	}
}

// fucks over cloudflare
func getPageWithUA(pageUrl string) ([]byte, error) {
	var err error

	var client = cycletls.Init()
	var resp cycletls.Response

	resp, err = client.Do(pageUrl, cycletls.Options{
		Body:      "",
		Ja3:       "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
		UserAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) Gecko/20100101 Firefox/87.0",
	}, "GET")
	if err != nil || resp.Status != http.StatusOK {
		return nil, errors.Join(errors.New("error downloading page"), err)
	}

	return []byte(resp.Body), err
}
