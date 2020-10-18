package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
)

type goodTooth struct {
	TotalPage int      `json:"total_page"`
	Clinics   []clinic `json:"clinics"`
}

type clinic struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Telphone string  `json:"telphone"`
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

const (
	//Host 主機
	Host = "http://wellness.hpa.gov.tw"

	//Path 路徑
	Path = "/App_Prog/MedicalList.aspx"

	//City 城市
	City = "01"

	//Town 鄉鎮
	Town = "0115"

	//Func 科別
	Func = "40"
)

func (gt goodTooth) New() *goodTooth {
	return &gt
}

func (gt *goodTooth) getPageNums(query string) {
	res, err := http.Get(query)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	doc.Find("div.contentB div.box1 div.page").Each(func(i int, s *goquery.Selection) {
		total := s.Find("table tr td a").Length()
		query, _ := s.Find("table tr td a").Eq(total - 1).Attr("href")
		u, err := url.Parse(query)
		if err != nil {
			log.Fatalln(err)
		}
		m, _ := url.ParseQuery(u.RawQuery)
		gt.TotalPage, err = strconv.Atoi(m["page"][0])
		if err != nil {
			log.Fatalln(err)
		}
	})
}

func (gt *goodTooth) getContent(query string) {
	res, err := http.Get(query)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	doc.Find("div.contentB div.box1 table#ContentPlaceHolder1_gvSearchList tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		name := s.Find("td").Eq(1).Text()
		arr := len(strings.Split(name, "診所"))
		if arr < 2 {
			return
		}
		id := s.Find("td").Eq(0).Text()
		telphone := s.Find("td").Eq(2).Text()
		address := s.Find("td").Eq(3).Text()
		lat, lng := gt.getLocation(address)
		c := clinic{id, name, telphone, address, lat, lng}
		gt.Clinics = append(gt.Clinics, c)
	})
}

func (gt *goodTooth) getLocation(address string) (float64, float64) {
	key := "AIzaSyC5Jt97fYuZTCEIeti-JT24ac5v3o9ceRY"
	query := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", address, key)
	res, err := http.Get(query)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	js, err := simplejson.NewJson([]byte(data))
	if err != nil {
		log.Fatalln(err)
	}
	lat, err := js.Get("results").GetIndex(0).Get("geometry").Get("location").Get("lat").Float64()
	if err != nil {
		log.Fatalln(err)
	}
	lng, err := js.Get("results").GetIndex(0).Get("geometry").Get("location").Get("lng").Float64()
	if err != nil {
		log.Fatalln(err)
	}
	return lat, lng
}

func (gt *goodTooth) writeJSON() {
	params := &gt
	b, err := json.Marshal(params)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data.json", b, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

func (gt *goodTooth) sync() {
	query := fmt.Sprintf("%s%s?cid=%s&tid=%s&ft=%s", Host, Path, City, Town, Func)
	gt.getPageNums(query)
	for i := 0; i < gt.TotalPage; i++ {
		queryPath := fmt.Sprintf("%s&page=%d", query, i)
		gt.getContent(queryPath)
	}
	gt.writeJSON()
}

func main() {
	gt := goodTooth{}.New()
	gt.sync()
}
