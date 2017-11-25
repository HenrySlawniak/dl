// Copyright (c) 2017 Henry Slawniak <https://henry.computer/>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dl

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/go-playground/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

var (
	userAgent = "dl v0.0.1"
	client    = &http.Client{}
)

// SetUserAgent will set the user agent to use with the http download client
func SetUserAgent(ua string) {
	userAgent = ua
}

// FileExists checks if the file already exists on disk
func FileExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

// GetBodyFromURL will return the body of the url
func GetBodyFromURL(u *url.URL, headers map[string]string, cookies *[]*http.Cookie) ([]byte, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	for _, c := range *cookies {
		req.AddCookie(c)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// GetRespFromURL will return the http.Response to a url
func GetRespFromURL(u *url.URL, headers map[string]string, cookies *[]*http.Cookie) (*http.Response, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {

		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	for _, c := range *cookies {
		req.AddCookie(c)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return client.Do(req)
}

// DownloadFile will download the url to fileloc
func DownloadFile(fileloc string, u *url.URL, headers map[string]string, cookies *[]*http.Cookie) (int64, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {

		return 0, err
	}

	req.Header.Set("User-Agent", userAgent)
	for _, c := range *cookies {
		req.AddCookie(c)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if !FileExists(fileloc) {
		// File isn't there, don't bother trying to avoid clobber
		return writeToFileFromURL(fileloc, u, headers, cookies)
	}

	head, err := client.Do(req)
	if err != nil {

		return 0, err
	}
	head.Body.Close()

	if head.Header.Get("Content-Length") == "" {
		// We didn't get the content length in the response
		return writeToFileFromURL(fileloc, u, headers, cookies)
	}

	length, err := strconv.ParseInt(head.Header.Get("Content-Length"), 10, 0)
	if err != nil {
		// content length can't be parsed, force dl
		return writeToFileFromURL(fileloc, u, headers, cookies)
	}

	f, err := os.Open(fileloc)
	if err != nil {

		return 0, err
	}

	stat, err := f.Stat()
	if err != nil {

		return 0, err
	}
	f.Close()

	if stat.Size() == length {
		fmt.Printf("Skipping %s (%s)\n", filepath.Base(fileloc), humanize.Bytes(uint64(length)))
		return 0, nil
	}

	return writeToFileFromURL(fileloc, u, headers, cookies)
}

func writeToFileFromURL(fileloc string, u *url.URL, headers map[string]string, cookies *[]*http.Cookie) (int64, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {

		return 0, err
	}

	req.Header.Set("User-Agent", userAgent)
	for _, c := range *cookies {
		req.AddCookie(c)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {

		return 0, err
	}
	defer resp.Body.Close()

	log.Debug(resp.StatusCode, ": ", resp.Status)
	log.Debug(resp.Header.Get("Content-Type"))

	length, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 0)
	if err != nil {
		log.Debugf("No Content-Length Header for %s", u.String())
	}

	var out *os.File

	if FileExists(fileloc) {
		out, err = os.OpenFile(fileloc, os.O_RDWR, os.FileMode(int(0775)))
		if err != nil {

			return 0, err
		}
		defer out.Close()
	} else {
		os.MkdirAll(filepath.Dir(fileloc), os.FileMode(0775))
		out, err = os.Create(fileloc)
		if err != nil {

			return 0, err
		}
		defer out.Close()
	}

	fmt.Printf("Downloading %s (%s)\n", filepath.Base(fileloc), humanize.Bytes(uint64(length)))

	return io.Copy(out, resp.Body)
}
