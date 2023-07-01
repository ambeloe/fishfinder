package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/antchfx/htmlquery"
	"github.com/buger/jsonparser"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var type1Regex = regexp.MustCompile("/us/en/browse/[0-9]{8}/")
var type2Regex = regexp.MustCompile("/us/en/products/[A-Z0-9]{8}/")

const apiUrl = "https://www.fishersci.com/us/en/catalog/service/browse/products/"

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
	var pageType int

	//parse url for future shenanigans
	baseUrl, err = url.Parse(*storeUrl)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error parsing url:", err)
		return 1
	}
	baseUrl.RawQuery = ""

	if type1Regex.MatchString(baseUrl.Path) {
		pageType = 1
	} else if type2Regex.MatchString(baseUrl.Path) {
		pageType = 2
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "unknown page type")
		return 1
	}

	switch pageType {
	//html page
	case 1:
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
	//js page -> json from api
	case 2:
		var numEntries = -1
		var offset int

		var tempI int64
		var tempS string

		*storeUrl = apiUrl + "?identifier=" + type2Regex.FindStringSubmatch(*storeUrl)[0][16:24]

		for {
			//fmt.Println(offset)
			resp, err = getPageWithUA(*storeUrl + "&offset=" + strconv.Itoa(offset))
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "error downloading json response:", err)
				return 1
			}

			if numEntries == -1 {
				tempI, err = jsonparser.GetInt(resp, "aggrRecordListSize")
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "unable to find number of results:", err)
					return 1
				}
				numEntries = int(tempI)
			}

			for i := 0; i < 30; i++ {
				//fmt.Println(offset + i)
				if e, _ := jsonparser.GetBoolean(resp, "productResults", "["+strconv.Itoa(offset+i)+"]", "hasOnlineSavings"); !e {
					continue
				}

				tempS, err = jsonparser.GetString(resp, "productResults", "["+strconv.Itoa(offset+i)+"]", "name")
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "couldn't get item name:", err)
				}
				if ipos := strings.Index(tempS, "<img"); ipos != -1 {
					tempS = tempS[:ipos-1]
				}
				tempS = strings.ReplaceAll(tempS, "&nbsp;", " ")
				tempS = strings.ReplaceAll(tempS, "&trade;", "™")

				fmt.Print(tempS + " | ")

				tempS, err = jsonparser.GetString(resp, "productResults", "["+strconv.Itoa(offset+i)+"]", "productUrl")
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "couldn't get item url:", err)
				}

				fmt.Println(baseUrl.Scheme + "://" + baseUrl.Host + tempS)
			}
			offset += 30
			if offset > numEntries {
				break
			}
		}
	default:
		_, _ = fmt.Fprintln(os.Stderr, "unknown page type")
		return 1
	}

	return 0
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
