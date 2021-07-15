// TODO make things concurrent
// TODO document everything
// TODO improve logging
// TODO use a progress bar?

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
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
	// parse URL
	u, err := url.Parse(os.Args[1])
	if err != nil {
		log.WithFields(log.Fields{
			"url": u,
		}).Error("unable to parse URL")
	}

	// check the URL for validity
	if !(strings.Contains(u.Hostname(), "4chan.org") ||
		strings.Contains(u.Hostname(), "4channel.org")) {
		log.WithFields(log.Fields{
			"url": u,
		}).Error("4chan-dl only supports downloading from 4chan")
	}
	parts := strings.Split(u.Path[1:], "/")
	board := parts[0]
	tid := parts[2]
	if parts[1] != "thread" {
		log.WithFields(log.Fields{
			"url": u,
		}).Error("URL is not a thread")
	}

	// make requests
	document, err := getPage(u)
	if err != nil {
		log.WithFields(log.Fields{
			"url": u,
		}).Error("could not get page")
	}

	// parse DOM and extract links
	links := document.Find(".thread .postContainer .file .fileText a")

	images := make([]string, links.Size())

	links.Each(func(i int, s *goquery.Selection) {
		imgpath, exists := s.Attr("href")
		if !exists {
			log.WithFields(log.Fields{
				"selection": s,
			}).Error("href attribute does not exist")
			// return
		}
		log.WithFields(log.Fields{
			"index": i,
			"text":  s.Text(),
		}).Infof("found %s", imgpath)

		// ignore the first two '//'
		images[i] = "https:" + imgpath
	})

	log.Debug(images)

	// create the thread dir
	if _, err := os.Stat(board + "/" + tid); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(board+"/"+tid, 0755)
		if err != nil {
			log.WithFields(log.Fields{
				"path": board + "/" + tid,
			}).Error(err)
		}
	}

	for _, imageURL := range images {
		// download images
		image, err := getImage(imageURL)
		if err != nil {
			log.WithFields(log.Fields{
				"image": imageURL,
			}).Error(err)
		}
		defer image.Close()

		// write images to disk
		path := fmt.Sprintf("%s/%s/%s", board, tid, path.Base(imageURL))
		file, err := os.Create(path)
		if err != nil {
			log.WithFields(log.Fields{
				"path": path,
			}).Error(err)
		}
		defer file.Close()

		_, err = io.Copy(file, image)
		if err != nil {
			log.WithFields(log.Fields{
				"path": path,
			}).Error(err)
		}

		log.New().WithFields(log.Fields{
			"file": file.Name(),
		}).Info("Done writing file.")
	}

	log.Info(u.Path)
	getPage(u)
}

func getImage(u string) (io.ReadCloser, error) {
	res, err := http.Get(u)
	if err != nil {
		log.WithFields(log.Fields{
			"url": u,
		}).Error(err)
	}
	return res.Body, err
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
			"url":        u,
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
