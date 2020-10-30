package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
	_ "github.com/joho/godotenv/autoload"
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
	Score    int     `json:"score"`
	Note     string  `json:"note"`
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

//New 建構式
func (gt goodTooth) New() *goodTooth {
	return &gt
}

//getPageNums 從國民健康署取得所有牙醫診所總分頁數
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

//getContent 從國民健康署取得所有牙醫診所
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
		address = strings.Split(address, "、")[0]
		address = strings.Split(address, "(")[0]
		strArr := strings.Split(address, "號")
		if len(strArr) == 1 {
			address = address + "號"
		}
		lat, lng := gt.getLocation(address)
		c := clinic{id, name, telphone, address, lat, lng, 0, ""}
		gt.Clinics = append(gt.Clinics, c)
	})
}

//取得附近是否有捷運站
func (gt *goodTooth) calcNearByMRT() {
	apiKey := os.Getenv("APIKey")
	nums := len(gt.Clinics)
	for i := 0; i < nums; i++ {
		clinic := gt.Clinics[i]
		dest := map[string]float64{"lat": 25.100800, "lng": 121.522310}
		query := fmt.Sprintf("https://router.hereapi.com/v8/routes?transportMode=pedestrian&origin=%f,%f&destination=%f,%f&return=travelSummary&units=imperial&lang=zh-tw&apiKey=%s", clinic.Lat, clinic.Lng, dest["lat"], dest["lng"], apiKey)
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
		length, err := js.Get("routes").GetIndex(0).Get("sections").GetIndex(0).Get("travelSummary").Get("baseDuration").Int()
		if err != nil {
			log.Fatalln(err)
		}
		if length <= 300 {
			gt.Clinics[i].Score += 5
			gt.Clinics[i].Note = "300m內有捷運站\n"
		} else {
			gt.Clinics[i].Score += 3
			gt.Clinics[i].Note = "300m外有捷運站\n"
		}
	}
}

//calcNearByClinics 取得附近診所
func (gt *goodTooth) calcNearByClinics() {
	apiKey := os.Getenv("APIKey")
	nums := len(gt.Clinics)
	for i := 0; i < nums; i++ {
		from := gt.Clinics[i]
		rooms := []string{}
		for j := 0; j < nums; j++ {
			if i == j {
				continue
			}
			to := gt.Clinics[j]
			query := fmt.Sprintf("https://router.hereapi.com/v8/routes?transportMode=pedestrian&origin=%f,%f&destination=%f,%f&return=travelSummary&units=imperial&lang=zh-tw&apiKey=%s", from.Lat, from.Lng, to.Lat, to.Lng, apiKey)
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
			length, err := js.Get("routes").GetIndex(0).Get("sections").GetIndex(0).Get("travelSummary").Get("baseDuration").Int()
			if err != nil {
				log.Fatalln(err)
			}
			if length <= 500 {
				rooms = append(rooms, to.Name)
			}
		}
		roomNums := len(rooms)
		if roomNums > 1 {
			gt.Clinics[i].Score++
		} else if roomNums == 1 {
			gt.Clinics[i].Score += 3
		} else {
			gt.Clinics[i].Score += 5
		}
		for j := 0; j < roomNums; j++ {
			gt.Clinics[i].Note += fmt.Sprintf("500m裡有%s\n", rooms[j])
		}
	}
}

//getLocaton 取得牙科經緯度
func (gt *goodTooth) getLocation(address string) (float64, float64) {
	apiKey := os.Getenv("APIKey")
	query := fmt.Sprintf("https://geocode.search.hereapi.com/v1/geocode?q=%s&apiKey=%s", url.QueryEscape(address), apiKey)
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
	lat, err := js.Get("items").GetIndex(0).Get("position").Get("lat").Float64()
	if err != nil {
		log.Fatalln(err)
	}
	lng, err := js.Get("items").GetIndex(0).Get("position").Get("lng").Float64()
	if err != nil {
		log.Fatalln(err)
	}
	return lat, lng
}

//writeJSON 寫入JSON檔供網頁端透過AJAX載入
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

//sync 同步JSON檔與網上資料
func (gt *goodTooth) sync() {
	query := fmt.Sprintf("%s%s?cid=%s&tid=%s&ft=%s", Host, Path, City, Town, Func)
	gt.getPageNums(query)
	for i := 0; i < gt.TotalPage; i++ {
		queryPath := fmt.Sprintf("%s&page=%d", query, i)
		gt.getContent(queryPath)
	}
	gt.calcNearByMRT()
	gt.calcNearByClinics()
	gt.writeJSON()
}

//main 主程式
func main() {
	gt := goodTooth{}.New()
	gt.sync()
}
