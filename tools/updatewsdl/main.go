package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb"
)

const (
	url = "http://www.onvif.org/Documents/Specifications.aspx"

	index   = 7
	timeout = 30
)

type wsdl struct {
	title string
	date  string
	url   string
}

type Wsdls struct {
	wl []wsdl
}

func NewWsdls() *Wsdls {
	wl := new(Wsdls)
	err := wl.getWsdl()
	if err != nil {
		log.Println(err)
		return nil
	}
	return wl
}

func (w *Wsdls) GetWsdlFiles() error {
	updateLog, err := os.OpenFile("../../wsdl/update.log", os.O_TRUNC|os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	checkErr(err)
	defer updateLog.Close()

	// fmt.Printf("总计：%d\n", len(w.wl))
	for index, wl := range w.wl {
		// fmt.Printf("%s -> %s -> %s\n\n", wl.Data, wl.Title, wl.Url)
		w.writeFile(wl.url)

		name := filepath.Base(wl.url)
		updateLog.WriteString(fmt.Sprintf("%-2d - %-21s - %s - %s - %s\r\n", index+1, name, wl.date, wl.title, wl.url))
	}
	updateLog.Sync()

	return nil
}

func (w *Wsdls) writeFile(wsurl string) {
	resp, err := http.Get(wsurl)
	defer resp.Body.Close()
	checkErr(err)
	if resp.StatusCode == http.StatusOK {
		wsName := filepath.Base(wsurl)
		if strings.Contains(wsurl, "ver20") {
			ext := filepath.Ext(wsName)
			wsName = strings.Split(wsName, ext)[0] + "2" + ext
		}
		os.MkdirAll("../../wsdl/", 0666)
		downFile, err := os.Create("../../wsdl/" + wsName)
		checkErr(err)
		defer downFile.Close()

		total, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		processBar(total, func(bar *pb.ProgressBar) {
			bar.Prefix(wsName + " -> ")
			dst := io.MultiWriter(downFile, bar)
			io.Copy(dst, resp.Body)
		})
	}
}

func (w *Wsdls) getWsdl() error {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return err
	}
	host := strings.Join([]string{
		doc.Url.Scheme,
		"://",
		doc.Url.Host}, "")
	wsdls := doc.
		Find("#dnn_ctr400_HtmlModule_lblContent ul").Eq(index).
		Find("li")

	wsdls.Each(func(i int, s *goquery.Selection) {
		result := strings.Split(s.Text(), "-")
		title := strings.TrimSpace(result[1])
		date := strings.TrimSpace(result[0])

		href, _ := s.Find("a").Attr("href")
		urls := fmt.Sprintf("%s%s", host, href)

		w.wl = append(w.wl, wsdl{title, date, urls})
	})

	return nil
}

func processBar(total int, fbar func(bar *pb.ProgressBar)) {
	bar := pb.New(total)
	bar.ShowFinalTime = true
	bar.SetUnits(pb.U_BYTES)

	bar.Start()
	fbar(bar)
	bar.Finish()
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	wl := NewWsdls()
	if wl == nil {
		return
	}

	fmt.Println("开始下载wsdl...")
	err := wl.GetWsdlFiles()
	if err != nil {
		log.Printf("GetWsdlFile -> %v\n", err)
		return
	}

	d := time.Duration(timeout * time.Second)
	log.Printf("wsdl 下载结束，%d秒后自动关闭窗口！", timeout)
	select {
	case <-time.After(d):
	}
}
