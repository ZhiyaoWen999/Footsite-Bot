package core

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	ayden "github.com/awxsam/adyen-footsites"
	"github.com/mattia-git/go-capmonster"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/tidwall/gjson"
)

var (
	sizeIDMap sync.Map
	a         = ayden.NewAdyen("A237060180D24CDEF3E4E27D828BDB6A13E12C6959820770D7F2C1671DD0AEF4729670C20C6C5967C664D18955058B69549FBE8BF3609EF64832D7C033008A818700A9B0458641C5824F5FCBB9FF83D5A83EBDF079E73B81ACA9CA52FDBCAD7CD9D6A337A4511759FA21E34CD166B9BABD512DB7B2293C0FE48B97CAB3DE8F6F1A8E49C08D23A98E986B8A995A8F382220F06338622631435736FA064AEAC5BD223BAF42AF2B66F1FEA34EF3C297F09C10B364B994EA287A5602ACF153D0B4B09A604B987397684D19DBC5E6FE7E4FFE72390D28D6E21CA3391FA3CAADAD80A729FEF4823F6BE9711D4D51BF4DFCB6A3607686B34ACCE18329D415350FD0654D")

	Queueit string
)

type Config struct {
	Capmonster string
	AntiCapcha string
	Delay      int
}

const (
	FtsQueueRecap   = "6LePTyoUAAAAADPttQg1At44EFCygqxZYzgleaKp"
	FtsRecapSiteKey = "6LccSjEUAAAAANCPhaM2c-WiRxCZ5CzsjR_vd8uX"
)

func getHash(input string, zeroCount int) (int, string) {
	zeros := strings.Repeat("0", zeroCount)
	for postfix := 0; ; postfix++ {
		str := input + strconv.Itoa(postfix)
		hash := sha256.New()
		hash.Write([]byte(str))
		encodedHash := hex.EncodeToString(hash.Sum(nil))
		if strings.HasPrefix(encodedHash, zeros) {
			return postfix, encodedHash
		}
	}
}

type Session struct {
	Site       string       `json:"site"`
	SiteSKU    string       `json:"siteSKU"`
	SizeID     string       `json:"size_ID"`
	Size       string       `json:"size"`
	SKU        string       `json:"sku"`
	UUID       string       `json:"uuid"`
	Client     *http.Client `json:"client"`
	CSRF       string       `json:"csrf"`
	JSESSIONID string       `json:"JSESSIONID"`
	Profile    *Profile     `json:"profile"`
	ProxyURL   *url.URL     `json:"proxy_url"`
	DataDome   string       `json:"datadome"`
	CartGuid   string       `json:"cart-guid"`
}

func (session *Session) InitSession(site string, siteSKU string, sku string, size string, proxy string, profile Profile) {
	session.Site = site
	session.SiteSKU = siteSKU
	session.SKU = sku
	session.Size = size
	u, _ := uuid.NewV4()
	session.UUID = u.String()
	session.ProxyURL, _ = url.Parse(proxy)

	session.Client = &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(session.ProxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   5 * time.Second,
	}

	session.Profile = &profile
}

func (session *Session) GenerateSession() error {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://www.%s.com/api/v3/session?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("Origin", fmt.Sprintf("https://www.%s.com", session.Site))
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com", session.Site))
	req.Header.Set("Origin", fmt.Sprintf("https://www.%s.com", session.Site))
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	resp, err := session.Client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		session.CSRF = string(body)[22:58]
		for _, v := range resp.Cookies() {
			if v.Name == "JSESSIONID" {
				session.JSESSIONID = v.Value

				return nil
			}
		}
	case 429:
		// TODO
		fmt.Println(errors.New("429"))
		return errors.New("429")
	case 403:
		// TODO
		fmt.Println(errors.New("403"))
		return errors.New("403")

	case 503:
		// TODO
		fmt.Println(errors.New("503"))
		return (errors.New("503"))
	}

	return errors.New("UNKNOWN")
}

func (session *Session) GetSizeID(c *capmonster.Client) error {

	sizeID, ok := sizeIDMap.Load(session.Size)

	if ok {
		session.SizeID = sizeID.(string)

		return nil
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://www.%s.com/zgw/product-core/v1/pdp/%s/sku/%s", session.Site, session.SiteSKU, session.SKU), nil)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.128 Safari/537.36")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("Origin", fmt.Sprintf("https://www.%s.com", session.Site))
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com", session.Site))
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID)

	resp, err := session.Client.Do(req)

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	// fmt.Println(resp.StatusCode)
	switch resp.StatusCode {

	case 200:

		sizeInformation := gjson.Get(string(body), "sizes")
		for _, v := range sizeInformation.Array() {

			if v.Get("size").String() == session.Size {

				session.SizeID = v.Get("productWebKey").String()
				sizeIDMap.Store(session.Size, session.SizeID)
				//fmt.Println("Get Size ID")
				return nil
			}
		}
		if session.SizeID == "" {
			return errors.New("unavailable size")
		}

	case 302, 301:
		fmt.Println("302")
		return errors.New("302")

	case 429:
		f, err := os.Create("Log\\SizeLog.txt")
		f.WriteString("Size Error \n" + "\n")
		f.WriteString("RespStatusCode: " + strconv.Itoa(resp.StatusCode) + "\n" + "\n")
		f.WriteString("Request URL: " + req.URL.String() + "\n" + "\n")
		f.WriteString("Request Header: \n")
		for name, value := range req.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}

		f.WriteString("\n" + "Response body:" + string(body) + "\n" + "\n")
		f.WriteString("Response Header: ")
		for name, value := range resp.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}
		if err != nil {
			f.Close()
		}
		defer f.Close()

	case 403:
		fmt.Println("403")
		return errors.New("403")
	case 503:
		fmt.Println("503")
		return errors.New("503")
	case 500:
		f, err := os.Create("Log\\SizeLog.txt")
		f.WriteString("Size Error \n" + "\n")
		f.WriteString("RespStatusCode: " + strconv.Itoa(resp.StatusCode) + "\n" + "\n")
		f.WriteString("Request URL: " + req.URL.String() + "\n" + "\n")
		f.WriteString("Request Header: \n")
		for name, value := range req.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}

		f.WriteString("\n" + "Response body:" + string(body) + "\n" + "\n")
		f.WriteString("Response Header: ")
		for name, value := range resp.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}
		if err != nil {
			f.Close()
		}
		defer f.Close()

	}
	return errors.New("UNKNOWN")

}

func (session *Session) AddToCart(c *capmonster.Client) error {

	data := map[string]interface{}{
		"productQuantity": 1,
		"productId":       session.SizeID,
	}

	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://www.%s.com/api/users/carts/current/entries?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payload))
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	//req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/product/~/%s.html", session.Site, session.SKU))
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("sec-ch-ua-platform", "Windows")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome)
	resp, err := session.Client.Do(req)
	if err != nil {
		return err
	}

	for _, v := range resp.Cookies() {
		switch v.Name {
		case "cart-guid":
			session.CartGuid = v.Value

		case "datadome":
			session.DataDome = v.Value

		default:
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))

	if err != nil {
		return err
	}
	resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return nil

	case 429:
		return errors.New("429")
	case 403:

		data := map[string]string{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return err
		}

		captchaURL := data["url"]
		u, _ := url.Parse(captchaURL)
		q := u.Query()
		if q.Get("cid") == "" || q.Get("initialCid") == "" || q.Get("hash") == "" || q.Get("referer") == "" || q.Get("t") == "bv" {
			return errors.New("FATAL")
		}

		fmt.Println(captchaURL)
		req, _ := http.NewRequest("GET", captchaURL, nil)
		resp, _ := session.Client.Do(req)

		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		var strgt string
		var strchall string
		gt := regexp.MustCompile(`(gt: \')(.*)(\')`)
		challenge := regexp.MustCompile(`(challenge: \')(.*)(\')`)

		gtmatchall := gt.FindAllStringSubmatch(string(body), -1)
		for _, v := range gtmatchall {
			strgt = v[2]

		}
		challengechall := challenge.FindAllStringSubmatch(string(body), -1)

		for _, v := range challengechall {
			strchall = v[2]

		}

		// Here you can change the capcha information to solve geetest , I just leave it here   from here...
		dataGee := map[string]interface{}{
			"gt":        strgt,
			"challenge": strchall,
		}
		payload, _ := json.Marshal(dataGee)
		reqGee, _ := http.NewRequest("POST", "", bytes.NewReader(payload))

		// to here...

		reqGee.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36")
		respGee, _ := session.Client.Do(reqGee)

		bodyGee, _ := ioutil.ReadAll(respGee.Body)
		defer respGee.Body.Close()
		// fmt.Println(string(bodyGee))
		// fmt.Println(respGee.StatusCode)

		token := gjson.Get(string(bodyGee), "token")
		seccode := gjson.Get(string(bodyGee), "seccode")
		challengeGee := gjson.Get(string(bodyGee), "challenge")
		ua := gjson.Get(string(bodyGee), "userAgent")
		// fmt.Println(token)
		// fmt.Println(seccode)
		// fmt.Println(challengeGee)
		// fmt.Println(ua)

		// Decode captchaURL

		// token, err := c.SendRecaptchaV2(captchaURL, FtsRecapSiteKey, time.Second*25)
		// if err != nil {

		// 	return err
		// }

		reqcap, _ := http.NewRequest("GET", "https://geo.captcha-delivery.com/captcha/check", nil)
		query := reqcap.URL.Query()
		query.Add("cid", q.Get("cid"))
		query.Add("icid", q.Get("initialCid"))
		query.Add("ccid", session.DataDome)
		query.Add("geetest-response-challenge", challengeGee.String())
		query.Add("geetest-response-validate", token.String())
		query.Add("geetest-response-seccode", seccode.String())
		query.Add("hash", q.Get("hash"))
		query.Add("ua", ua.String())
		query.Add("referer", q.Get("referer"))
		query.Add("parent_url", captchaURL)
		query.Add("x-forwarded-for", "")
		reqcap.Header.Set("accept", "*/*")
		reqcap.Header.Set("accept-encoding", "gzip, deflate, br")
		reqcap.Header.Set("Content-Type", " application/x-www-form-urlencoded; charset=UTF-8")
		reqcap.Header.Set("Connection", " keep-alive")
		reqcap.Header.Set("Host", "geo.captcha-delivery.com")
		reqcap.Header.Set("user-agent", ua.String())
		reqcap.Header.Set("Referer", captchaURL)
		reqcap.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
		reqcap.Header.Set("Sec-Fetch-Site", "same-origin")
		reqcap.Header.Set("Sec-Fetch-Mode", "cors")
		reqcap.Header.Set("Sec-Fetch-Dest", "empty")
		reqcap.Header.Set("sec-ch-ua-platform", "Windows")
		reqcap.Header.Set("sec-ch-ua-mobile", "?0")
		reqcap.Header.Set("Cookie", "datadome="+session.DataDome)
		reqcap.URL.RawQuery = query.Encode()
		respcap, _ := session.Client.Do(reqcap)

		bodycap, _ := ioutil.ReadAll(respcap.Body)

		defer resp.Body.Close()
		// fmt.Println(respcap.StatusCode)
		// fmt.Println(string(bodycap))

		if resp.StatusCode == 200 {
			// The dd cookie is in response body :)
			dd := make(map[string]interface{})
			err = json.Unmarshal(bodycap, &dd)
			if err != nil {
				return err
			}
			cookie, ok := dd["cookie"]
			if ok {
				session.DataDome = cookie.(string)[9:115]

			}
			//fmt.Println(session.DataDome)
			return errors.New("DONE")
		}

		return errors.New("403")
	case 503:

		return errors.New("503")
	}
	return errors.New("UNKNOWN")

}

func (session *Session) SubmitEmail() error {
	req, _ := http.NewRequest("PUT", fmt.Sprintf("https://www.%s.com/api/users/carts/current/email/%s?timestamp=%v", session.Site, session.Profile.Email, time.Now().UnixNano()/int64(time.Millisecond)), nil)

	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout.html", session.Site))
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)
	resp, err := session.Client.Do(req)

	if err != nil {
		return err
	}

	ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:

		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value

			default:

			}
		}
		return nil
	case 429:
		// TODO
		return errors.New("429")
	case 403:
		// TODO
		return errors.New("403")
	case 503:
		// TODO
		return errors.New("503")
	case 400:
		f, err := os.Create("Log\\EmailLog.txt")
		f.WriteString("Size Error \n" + "\n")
		f.WriteString("RespStatusCode: " + strconv.Itoa(resp.StatusCode) + "\n" + "\n")
		f.WriteString("Request URL: " + req.URL.String() + "\n" + "\n")
		f.WriteString("Request Header: \n")
		for name, value := range req.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}

		f.WriteString("Response Header: ")
		for name, value := range resp.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}
		if err != nil {
			f.Close()
		}
		defer f.Close()
		return errors.New("400")
	}
	return errors.New("UNKNOWN")

}

func (session *Session) SubmitShipping() error {
	isState, _ := makekey(usc, session.Profile.State)

	data := map[string]interface{}{
		"shippingAddress": map[string]interface{}{
			"billingAddress": false,
			"companyName":    "",
			"country": map[string]string{
				"isocode": "US",
				"name":    "United States",
			},
			"defaultAddress": false,
			"email":          session.Profile.Email,
			"firstName":      session.Profile.FirstName,
			"id":             nil,
			"isFPO":          false,
			"lastName":       session.Profile.LastName,
			"line1":          session.Profile.Line1,
			"line2":          session.Profile.Line2,
			"phone":          session.Profile.Phone,
			"postalCode":     session.Profile.PostalCode,
			"recordType":     "S",
			"region": map[string]string{
				"countryIso":   "US",
				"isocode":      "US-" + isState,
				"isocodeShort": isState,
				"name":         session.Profile.State,
			},
			"regionFPO":            nil,
			"saveInAddressBook":    false,
			"setAsBilling":         true,
			"setAsDefaultBilling":  false,
			"setAsDefaultShipping": false,
			"shippingAddress":      true,
			"town":                 strings.ToUpper(session.Profile.City),
			"type":                 "default",
			"visibleInAddressBook": false,
			"LoqateSearch":         "",
		},
	}
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://www.%s.com/api/users/carts/current/addresses/shipping?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payload))
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout.html", session.Site))
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)
	resp, err := session.Client.Do(req)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {

	case 200, 201:
		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value
			default:
			}
		}

		return nil

	case 429:
		fmt.Println("429")
		return errors.New("429")
	case 403:
		// TODO
		fmt.Println("403")
		return errors.New("403")
	case 503:
		// TODO
		fmt.Println("503")
		return errors.New("503")
	case 400:
		f, err := os.Create("Log\\ShippingLog.txt")
		f.WriteString("Shipping Error \n" + "\n")
		f.WriteString("RespStatusCode: " + strconv.Itoa(resp.StatusCode) + "\n" + "\n")
		f.WriteString("Request URL: " + req.URL.String() + "\n" + "\n")
		f.WriteString("Request Header: \n")
		for name, value := range req.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}

		f.WriteString("\n" + "Response body:" + string(body) + "\n" + "\n")
		f.WriteString("Response Header: ")
		for name, value := range resp.Header {
			f.WriteString("\t" + name + ":")
			f.WriteString("\t" + value[0] + "\n")
		}
		if err != nil {
			f.Close()
		}
		defer f.Close()

	}
	return errors.New("UNKNOWN")
}

func (session *Session) SubmitBilling() error {
	isState, _ := makekey(usc, session.Profile.State)
	data := map[string]interface{}{
		"setAsDefaultBilling":  false,
		"setAsDefaultShipping": false,
		"firstName":            session.Profile.FirstName,
		"lastName":             session.Profile.LastName,
		"email":                session.Profile.Email,
		"phone":                session.Profile.Phone,
		"country": map[string]string{
			"isocode": "US",
			"name":    "United States",
		},
		"id":                nil,
		"setAsBilling":      true,
		"saveInAddressBook": false,
		"region": map[string]string{
			"countryIso":   "US",
			"isocode":      "US-" + isState,
			"isocodeShort": isState,
			"name":         session.Profile.State,
		},
		"type":            "default",
		"LoqateSearch":    "",
		"line1":           session.Profile.Line1,
		"line2":           session.Profile.Line2,
		"postalCode":      session.Profile.PostalCode,
		"town":            strings.ToUpper(session.Profile.City),
		"regionFPO":       nil,
		"shippingAddress": true,
		"recordType":      "H",
	}

	payloadBytes, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", fmt.Sprintf("https://www.%s.com/api/users/carts/current/set-billing?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payloadBytes))
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout.html", session.Site))
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)
	resp, err := session.Client.Do(req)
	if err != nil {
		return err
	}
	ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {

	case 200, 201:

		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value
			default:
			}
		}

		return nil

	case 429:
		return errors.New("429")
	case 403:
		// TODO
		return errors.New("403")
	case 503:
		// TODO
		return errors.New("503")
	}
	return errors.New("UNKNOWN")
}

func (session *Session) GiftCard(svcPIN string, svcNumber string) error {
	data := map[string]interface{}{
		"svcPIN":    svcPIN,
		"svcNumber": svcNumber,
	}
	payload, _ := json.Marshal(data)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("https://www.%s.com/api/users/carts/current/add-giftcard?timestamp=%v ", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payload))
	req.Header.Set("Host", "www.footlocker.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Length", "49")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("Sec-ch-ua-mobile", "?0")
	req.Header.Set("Accept-encoding", "gzip, deflate, br")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout", session.Site))
	req.Header.Set("Origin", fmt.Sprintf("https://www.%ss.com", session.Site))
	req.Header.Set("Accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)
	resp, err := session.Client.Do(req)

	if err != nil {
		return err
	}
	ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200, 201:

		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value
			default:

			}
		}
		return nil
	case 400:
		return errors.New("400")
	case 429:

		return errors.New("429")
	case 403:

		return errors.New("403")
	case 503:

		return errors.New("503")
	}
	return errors.New("UNKNOWN")
}

/*POST https://www.footlocker.com/api/users/carts/current/add-giftcard?timestamp=1626422136479 HTTP/1.1

{"svcPIN":"12323123","svcNumber":"1231241223414"}*/

func (session *Session) PickPerson() error {
	data := map[string]interface{}{
		"email":     session.Profile.Email,
		"firstName": session.Profile.FirstName,
		"lastName":  session.Profile.LastName,
	}

	payload, _ := json.Marshal(data)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("https://www.%s.com/api/users/carts/current/pickperson?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-encoding", "gzip, deflate, br")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout", session.Site))
	req.Header.Set("Origin", fmt.Sprintf("https://www.%ss.com", session.Site))
	req.Header.Set("Accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)
	resp, err := session.Client.Do(req)

	if err != nil {
		return err
	}
	ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200, 201:

		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value
			default:

			}
		}
		return nil
	case 400:
		return errors.New("400")
	case 429:

		return errors.New("429")
	case 403:

		return errors.New("403")
	case 503:

		return errors.New("503")
	}
	return errors.New("UNKNOWN")
}

func (session *Session) SubmitOrder() error {
	cc, en, ey, cvv, _ := a.EncryptCreditcardDetails(session.Profile.CCNumber, session.Profile.ExpMonth, session.Profile.ExpYear, session.Profile.Cvv)
	data := map[string]interface{}{
		"preferredLanguage":     "en",
		"termsAndCondition":     false,
		"deviceId":              "0400tyDoXSFjKeoNf94lis1ztioT9A1DShgAnrp/XmcfWoVVgr+Rt2dAZPhMS97Z4yfjSLOS3mruQCzk1eXuO7gGCUfgUZuLE2xCJiDbCfVZTGBk19tyNs7g8zV85QpvmtF/PiH81LzIHY89C7pjSl/JxUN13n2vmAeykQgdlVeDidx1G2mpGiKJ4Ao5VNMvaDXf7E1Pf46IvXtYdEyMOakFzprLKk3u1s0Iq1jEc21Hw6sowi9Jf88gkkjXzk77ILZ/eUsQ7RNrLro1kTKIs1496YkpIh3A707lm2e25SQbo1OuF8qR6VxrbC1wRHKPI15Qt45gqkMMYYfmY1XpDGBtyepPcth3j49FbUw/Y7k8g+pI+pjSFsqkUH6/N04I+t4CSViaqSM75eAyoDLQV8tjoO5wWTLXZT8tb6o9WQLwaJrb8RzWdzZK4bduoeAODFWWn+P0HNw3kOx04hiAcL/dBjV9nBJW1Y3JLACqjtcRnAvzJ1F0W5Ivre9RJsn4u3PdHf7WUtiodkTscXNLCbvmpFoNB0kg9fZ5lAWxTN4DFe0pQs/EGzmZL5NHY4PWj2UUP1K1FVKPNfm2EMlMtbRnNwQxSxzPuRtycb0H5IjHkCcKSs4KVYKB4i89PjJjT+QZ/Uoiy6yqSntHDo0vYYDC35oMiutO9ehTdLDs+1JC1RHBTwTVw5JVoFEa2lLiexN4OaKgIv3sbV/sTU9/joi9e1h0TIw5qQXOmssqTe7WzQirWMRzbUfDqyjCL0l/zyCSSNfOTvsgtn95SxDtE2suujWRMoizXj3piSkiHcDvTuWbZ7blJBujU64XypHpXGtsLXBEco8jXlC3jmCqQwxhh+ZjVekMYG3JYXyFHBbsmyyzb0vSwWRA+eSMeWgCk0M8kgQVobtOy5nu7pRmssxc1n3Rzh1ozHM1w5rNRhjMFvlSwejOX5dhKGYrsu13rc4RCbryu9G8AaTIauRtCQwBH44X08jLtlj16MHb+jKUckBfu00yz9/YQl2DyjiNQ5FMWmFLFbkufQOZu721S+SsYPbPQSWnaPfVhKRSjB4oZ28MlyI9q4+qN2hV2SY+Vz58fqYAndlMe7bZ0GxLd+c4lDeGG4xIicZNyoU7LduN6ZrcE4nZ3meVnnb6dxDioksejgtHHqqE28hv5t2asbd8NQZ+i8aFjp2buuK69Oy8ERmOBnl5tj9a8Yrr+l3H+CrCxGFawmFFcIAKFEZFQI9MzuB56CKvT5VTxedGLMijsMcQRH9L1eBfqylVWCPnhiZw",
		"cartId":                session.CartGuid,
		"encryptedCardNumber":   cc,
		"encryptedExpiryMonth":  en,
		"encryptedExpiryYear":   ey,
		"encryptedSecurityCode": cvv,
		"paymentMethod":         "CREDITCARD",
		"returnUrl":             "https://www.footlocker.com/adyen/checkout",
		"browserInfo": map[string]interface{}{
			"screenWidth":    1440,
			"screenHeight":   900,
			"colorDepth":     30,
			"userAgent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.128 Safari/537.36",
			"timeZoneOffset": 240,
			"language":       "en-US",
			"javaEnabled":    false,
		},
	}
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://www.%s.com/api/v2/users/orders?timestamp=%v", session.Site, time.Now().UnixNano()/int64(time.Millisecond)), bytes.NewReader(payload))
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/checkout.html", session.Site))
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("X-Fl-Request-Id", session.UUID)
	req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
	req.Header.Set("X-Csrf-Token", session.CSRF)
	req.Header.Set("X-Fl-Productid", session.SizeID)
	req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID+"; "+"datadome="+session.DataDome+";"+"cart-guid="+session.CartGuid)

	resp, err := session.Client.Do(req)
	if err != nil {
		return err
	}
	ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200, 201:
		// SUCCESS
		for _, v := range resp.Cookies() {
			switch v.Name {
			case "datadome":
				session.DataDome = v.Value
			default:
				// Do Nothing
			}
		}
		return nil
	case 429:
		// TODO
		return errors.New("429")
	case 403:
		// TODO
		return errors.New("403")
	case 503:
		// TODO
		return errors.New("503")
	}
	return errors.New("payment decline")

}
