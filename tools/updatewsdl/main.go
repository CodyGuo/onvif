package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb"
)

const (
	url   = "http://www.onvif.org/Documents/Specifications.aspx"
	index = 7
)

type WSDL struct {
	Title string
	Data  string
	Url   string
}

func (ws *WSDL) queryParse() <-chan WSDL {
	doc, err := goquery.NewDocument(url)
	checkErr("newDocument", err)

	var scheme string
	if doc.Url.Scheme == "http" {
		scheme = "http://"
	} else {
		scheme = "https://"
	}
	urlHost := strings.Join([]string{scheme, doc.Url.Host}, "")

	out := make(chan WSDL)
	var wg sync.WaitGroup
	wsdls := doc.Find("#dnn_ctr400_HtmlModule_lblContent ul").Eq(index).Find("li")
	wsdls.Each(func(i int, s *goquery.Selection) {
		result := strings.Split(s.Text(), "-")
		title := strings.TrimSpace(result[1])
		data := strings.TrimSpace(result[0])
		href, _ := s.Find("a").Attr("href")
		urls := fmt.Sprintf("%s%s", urlHost, href)

		wg.Add(1)
		go func() {
			out <- WSDL{title, data, urls}
			wg.Done()
		}()

	})
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func getWsdlFile(ws string) {
	names := strings.Split(ws, "/")
	wsName := names[len(names)-1]
	if strings.Contains(wsName, "=") {
		names := strings.Split(wsName, "=")
		wsName = names[len(names)-1]
	}
	resp, err := http.Get(ws)
	defer resp.Body.Close()
	checkErr("getWsdlFile", err)
	if resp.StatusCode == http.StatusOK {
		downFile, err := os.Create("../../wsdl/" + wsName)
		checkErr("create WSDL", err)
		defer downFile.Close()
		i, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
		sourceSiz := int64(i)
		source := resp.Body

		bar := pb.New(int(sourceSiz)).SetUnits(pb.U_BYTES).Prefix(wsName + " ")

		bar.Start()
		bar.ShowFinalTime = true
		bar.SetMaxWidth(80)

		writer := io.MultiWriter(downFile, bar)
		io.Copy(writer, source)
		bar.Finish()
	}
}

func main() {
	ws := new(WSDL)

	query := ws.queryParse()
	var wsUrls []WSDL
	for o := range query {
		wsUrls = append(wsUrls, o)
	}

	for _, ws := range wsUrls {
		// fmt.Printf("%s -> %s -> %s\n\n", ws.Data, ws.Title, ws.Url)
		getWsdlFile(ws.Url)
	}

	const timeout = 10
	d := time.Duration(timeout * time.Second)
	fmt.Printf("下载结束，%d秒后自动关闭窗口！", timeout)
	select {
	case <-time.After(d):
	}
}

func checkErr(info string, err error) {
	if err != nil {
		log.Fatalf("%s -> %s\n", info, err)
	}
}
