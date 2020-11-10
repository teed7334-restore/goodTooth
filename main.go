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
	TotalPage     int            `json:"total_page"`
	Clinics       []clinic       `json:"clinics"`
	NearByMRTs    []nearByMRT    `json:"nearByMRTs"`
	NearByClinics []nearByClinic `json:"nearByClinics"`
	NearBySchools []nearBySchool `json:"nearBySchools"`
	Schools       []school       `json:"schools"`
}

type clinic struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Telphone string  `json:"telphone"`
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type nearByMRT struct {
	Distance int `json:"distance"`
}

type nearByClinic struct {
	Distance map[int]int `json:"distance"`
}

type nearBySchool struct {
	Distance map[int]int `json:"distance"`
}

type school struct {
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
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

	//EarthRadius 地球半徑
	EarthRadius = 6371
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

//getClinics 從國民健康署取得所有牙醫診所
func (gt *goodTooth) getClinics(query string) {
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
		c := clinic{id, name, telphone, address, lat, lng}
		gt.Clinics = append(gt.Clinics, c)
	})
}

//getSchools 從教育局取得所有學校
func (gt *goodTooth) getSchools(query string) {
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
	doc.Find("table").Eq(6).Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		name := s.Find("td").Eq(0).Text()
		address := s.Find("td").Eq(1).Text()
		arr := strings.Split(address, "]")
		address = arr[len(arr)-1]
		lat, lng := gt.getLocation(address)
		c := school{name, address, lat, lng}
		gt.Schools = append(gt.Schools, c)
	})
}

//calcNearByMRT 計算與捷運站距離
func (gt *goodTooth) calcNearByMRT() {
	apiKey := os.Getenv("APIKey")
	nums := len(gt.Clinics)
	dest := map[string]float64{"lat": 25.100800, "lng": 121.522310}
	gt.NearByMRTs = make([]nearByMRT, 0)
	for i := 0; i < nums; i++ {
		clinic := gt.Clinics[i]
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
		gt.NearByMRTs = append(gt.NearByMRTs, nearByMRT{0})
		gt.NearByMRTs[i].Distance = length
	}
}

//calcNearByClinics 計算與附近診所距離
func (gt *goodTooth) calcNearByClinics() {
	apiKey := os.Getenv("APIKey")
	nums := len(gt.Clinics)
	gt.NearByClinics = make([]nearByClinic, 0)
	for i := 0; i < nums; i++ {
		from := gt.Clinics[i]
		nbcs := make(map[int]int)
		for j := 0; j < nums; j++ {
			if i == j { //等於自已跳過
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
			nbcs[j] = length
		}
		gt.NearByClinics = append(gt.NearByClinics, nearByClinic{})
		gt.NearByClinics[i].Distance = nbcs
	}
}

//calcNearBySchools 計算與附近學校距離
func (gt *goodTooth) calcNearBySchools() {
	apiKey := os.Getenv("APIKey")
	nums := len(gt.Clinics)
	gt.NearBySchools = make([]nearBySchool, 0)
	for i := 0; i < nums; i++ {
		from := gt.Clinics[i]
		nbcs := make(map[int]int)
		schoolNums := len(gt.Schools)
		for j := 0; j < schoolNums; j++ {
			to := gt.Schools[j]
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
			nbcs[j] = length
		}
		gt.NearBySchools = append(gt.NearBySchools, nearBySchool{})
		gt.NearBySchools[i].Distance = nbcs
	}
}

//getLocaton 取得經緯度
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

//writeClinicsToJson 將診所資料寫入JSON
func (gt *goodTooth) writeClinicsToJson() {
	clinics := &gt.Clinics
	byteValue, err := json.Marshal(clinics)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data/clinics.json", byteValue, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

//writeNearByMRTToJson 將捷運距離寫入JSON
func (gt *goodTooth) writeNearByMRTsToJson() {
	nearByMRT := &gt.NearByMRTs
	byteValue, err := json.Marshal(nearByMRT)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data/nearByMRTs.json", byteValue, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

//writeNearByClinicsToJSON 將各診所之間的距離寫入JSON
func (gt *goodTooth) writeNearByClinicsToJSON() {
	nearByClinics := &gt.NearByClinics
	byteValue, err := json.Marshal(nearByClinics)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data/nearByClinics.json", byteValue, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

//writeSchoolsToJSON 將學校資料寫入JSON
func (gt *goodTooth) writeSchoolsToJSON() {
	schools := &gt.Schools
	byteValue, err := json.Marshal(schools)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data/schools.json", byteValue, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

//writeNearBySchoolsToJSON 將與學校距離寫入JSON
func (gt *goodTooth) writeNearBySchoolsToJSON() {
	nearBySchools := &gt.NearBySchools
	byteValue, err := json.Marshal(nearBySchools)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("./data/nearBySchools.json", byteValue, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}

//writeJSON 寫入JSON檔供網頁端透過AJAX載入
func (gt *goodTooth) writeJSON() {
	gt.writeClinicsToJson()
	gt.writeNearByMRTsToJson()
	gt.writeNearByClinicsToJSON()
	gt.writeSchoolsToJSON()
	gt.writeNearBySchoolsToJSON()
}

//getClinicFromWeb 取得診所資料來自爬蟲
func (gt *goodTooth) getClinicFromWeb() {
	query := fmt.Sprintf("%s%s?cid=%s&tid=%s&ft=%s", Host, Path, City, Town, Func)
	gt.getPageNums(query)
	for i := 0; i < gt.TotalPage; i++ {
		queryPath := fmt.Sprintf("%s&page=%d", query, i)
		gt.getClinics(queryPath)
	}
}

//getSchoolFromWeb 取得學校資料來自爬蟲
func (gt *goodTooth) getSchoolFromWeb() {
	query := "https://www.doe.gov.taipei/News_Content.aspx?n=026199D6B5AC5A6A&sms=DDAA880EFAADF5F3&s=7472A783D2FDD6F7#shilin"
	gt.getSchools(query)
}

//getClinicFromFile 取得診所資料來自檔案
func (gt *goodTooth) getClinicFromFile() {
	jsonFile, err := os.Open("./data/clinics.json")
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(byteValue, &gt.Clinics)
}

//getSchoolFromFile 取得學校資料來自檔案
func (gt *goodTooth) getSchoolFromFile() {
	jsonFile, err := os.Open("./data/schools.json")
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(byteValue, &gt.Schools)
}

//getNearByMRTFromFile 取得捷運距離來自檔案
func (gt *goodTooth) getNearByMRTFromFile() {
	jsonFile, err := os.Open("./data/nearByMRTs.json")
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(byteValue, &gt.NearByMRTs)
}

//getNearByClinicFromFile 取得附近診所距離來自檔案
func (gt *goodTooth) getNearByClinicFromFile() {
	jsonFile, err := os.Open("./data/nearByClinics.json")
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(byteValue, &gt.NearByClinics)
}

//getNearBySchoolFromFile 取得附近學校距離來自檔案
func (gt *goodTooth) getNearBySchoolFromFile() {
	jsonFile, err := os.Open("./data/nearBySchools.json")
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln(err)
	}
	json.Unmarshal(byteValue, &gt.NearBySchools)
}

//sync 同步資料
func (gt *goodTooth) sync() {
	//gt.getClinicFromWeb()
	gt.getClinicFromFile()
	//gt.calcNearByMRT()
	gt.getNearByMRTFromFile()
	//gt.calcNearByClinics()
	gt.getNearByClinicFromFile()
	//gt.getSchoolFromWeb()
	gt.getSchoolFromFile()
	//gt.calcNearBySchools()
	gt.getNearBySchoolFromFile()
	gt.writeJSON()
}

//main 主程式
func main() {
	gt := goodTooth{}.New()
	gt.sync()
}
