# goodTooth
  本程式用來做為民國109年陽明大學醫務管理研究所學分班
  
  Quantitative Methods in Health Care Management，

  Chapter 3. Decision Making in Healthcare
  
  作業參考用，之後看需求決定要不要繼續開發下去

## 資料結構
---
./data 後端整理後資料檔

./env.swp 設定檔

./index.html 網頁端主程式

./map.html 網頁端地圖

./main.go 生成data.json之爬蟲與分析程式

./go.mod Golang有用到的Packages

## 操作說明
1. 請先將.env.swp置換成.env
2. 將.env當中的APIKey給補上，可以註冊Here Map API取得它
3. 運行main.go
4. 運行網頁伺服器(我這邊是用 php -S localhost:8080 來當作網頁伺服器)，開啟index.html

## 運行原理
---
本程式是透過Golang程式做為爬蟲，先去國民健康署抓到所有士林區牙醫診所列表之後，透過Here Map API，取得各對應之經緯度，再依診所評分表所依據之情況，透過爬蟲抓回所需之資料去做分析判斷，最後寫入到./data資料夾當中，再由map.html與index.html這二隻網頁檔，透過AJAX所讀取，透過Google Map將所有士林區牙醫診所標記在地圖上，且提供有效之評分依據

本程式爬蟲目前沒有用到並發與並行，如果有要繼續開發的話，會將程式改用並發與並行，如有要修改本程式，請參照我之前寫過之搶標機器人，裡面有並發並行之案例

[https://github.com/teed7334-restore/master](https://github.com/teed7334-restore/master)