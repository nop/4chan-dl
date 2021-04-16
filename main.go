package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

var client = &http.Client{}

const user_agent = "github.com/nop/4chan-dl"

func init() {
	// https://stackoverflow.com/a/47515580
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		lvl = "info"
	}
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	log.SetLevel(ll)
}

func main() {
	link, err := url.Parse(os.Args[1])
	if err != nil {
		log.WithFields(log.Fields{
			"url": link,
		}).Fatal("unable to parse URL")
	}

	pth := strings.Split(link.Path, "/")[1:]
	pthlog := log.WithFields(log.Fields{
		"url":  link,
		"path": pth,
	})
	if len(pth) != 3 {
		pthlog.WithFields(log.Fields{
			"expected_len": "3",
			"len":          len(pth),
		}).Fatal("invalid path length")
	}

	if pth[1] != "thread" {
		pthlog.WithFields(log.Fields{
			"expected": "thread",
			"actual":   pth[1],
		}).Warn("unexpected pth[1]")
	}

	board := pth[0]
	tid := pth[2]
	log.Info("Board: ", board)
	log.Info("Thread ID: ", tid)

	// make file
	outfile, err := os.Create("outfile.html")
	if err != nil {
		log.Fatal(err)
	}
	defer outfile.Close()

	doc, err := getPage(link)
	if err != nil {
		log.Fatal(err)
	}

	rootnode := doc.Find("form#delform div.board div.thread")
	posts := rootnode.Find(".postContainer")
	posts.Each(func(i int, s *goquery.Selection) {
		log.Info(i, *s)
	})
	log.Info("posts: ", posts)

}

// FIXME no timeout
func getPage(u *url.URL) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req.Header.Set("User-Agent", user_agent)

	res, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   u,
		}).Error("GET request error occurred")
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.WithFields(log.Fields{
			"statuscode": res.StatusCode,
			"status":     res.Status,
		}).Error("status code error")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return doc, err
}
