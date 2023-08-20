package core

import
fmt.Println(resp.StatusCode)
for _, v := range resp.Cookies() {
	switch v.Name {
	case "Queue-it":
		Queueit = v.Value
	default:
	}
}

redirectURL := resp.Request.URL

u, _ := url.Parse(redirectURL.String())
q := u.Query()
//c := strings.NewReader(q.Get("c"))

//t := strings.NewReader(q.Get("t"))
//cid := strings.NewReader(q.Get("cid"))

if q.Get("c") == "" || q.Get("e") == "" || q.Get("t") == "" || q.Get("cid") == "" {
	return errors.New("FATAL")
}

token, err := c.SendRecaptchaV2(redirectURL.String(), FtsQueueRecap, time.Second*25)
if err != nil {
	return err
}
//return errors.New("DONE")

fmt.Println("After get token")
data := map[string]interface{}{
	"challengeType": "recaptcha-invisible",
	"customerId":    "footlocker",
	"eventId":       q.Get("e"),
	"sessionId":     token,
	"version":       "5",
}

payload, _ := json.Marshal(data)
//send the cap token to verify
reqcap, _ := http.NewRequest("GET", "https://footlocker.queue-it.net/challengeapi/verify", bytes.NewReader(payload))
reqcap.Header.Set("Host", "footlocker.queue-it.net") //
reqcap.Header.Set("Connection", "keep-alive")
reqcap.Header.Set("Content-Length", "625")
reqcap.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
reqcap.Header.Set("X-Requested-With", "XMLHttpRequest")
reqcap.Header.Set("sec-ch-ua-mobile", "?0")
reqcap.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
reqcap.Header.Set("Content-Type", "application/json")
reqcap.Header.Set("Origin", "https://footlocker.queue-it.net")
reqcap.Header.Set("sec-fetch-site", "same-origin")
reqcap.Header.Set("sec-fetch-mode", "cors")
reqcap.Header.Set("sec-fetch-dest", "empty")
reqcap.Header.Set("accept-encoding", "gzip, deflate, br")
reqcap.Header.Set("Referer", redirectURL.String())
reqcap.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")

resp, err := session.Client.Do(reqcap)

if err != nil {
	return err
}

body, err := ioutil.ReadAll(resp.Body)

if err != nil {
	return err
}

defer resp.Body.Close()

switch resp.StatusCode {

case 429:
	// TODO
	fmt.Println(errors.New("429"), resp.Request.URL)
	return errors.New("429")
case 403:
	// TODO
	fmt.Println(errors.New("403"), resp.Request.URL)

	return errors.New("403")

case 503:
	// TODO
	fmt.Println(errors.New("503"))
	return (errors.New("503"))

case 200:

	challengeTypeC := gjson.Get(string(body), "sessionInfo.challengeType").String()
	checksumC := gjson.Get(string(body), "sessionInfo.checksum").String()
	sessionIdC := gjson.Get(string(body), "sessionInfo.sessionId").String()
	scourceIpC := gjson.Get(string(body), "sessionInfo.scourceIp").String()
	timestampC := gjson.Get(string(body), "sessionInfo.timestamp").String()
	versionC := gjson.Get(string(body), "sessionInfo.version").String()
	// request pow
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://footlocker.queue-it.net/challengeapi/pow/challenge/%s", Queueit), nil)
	req.Header.Set("Host", "footlocker.queue-it.net") //
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Length", "0")
	req.Header.Set("powTag-UserId", Queueit)
	req.Header.Set("powTag-EventId", q.Get("e"))
	req.Header.Set("powTag-CustomerId", "footlocker")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://footlocker.queue-it.net")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("Referer", redirectURL.String())
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Cookie", fmt.Sprintf("Queue-it=u=", Queueit))

	respChallenge, _ := session.Client.Do(req)

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	switch respChallenge.StatusCode {

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

	case 200:
		meta := gjson.Get(string(body), "meta").String()
		input := gjson.Get(string(body), "parameters.input").String()
		zeroCount := gjson.Get(string(body), "parameters.zeroCount").String()
		sessionId := gjson.Get(string(body), "sessionId").String()

		zeroCountInt, _ := strconv.Atoi(zeroCount)

		postfix, hash := getHash(input, zeroCountInt)

		proof := map[string]interface{}{
			"userId":    Queueit,
			"meta":      meta,
			"sessionId": sessionId,
			"solution": map[string]interface{}{
				"postfix": postfix,
				"hash":    hash,
			},
			"tags": []string{fmt.Sprintf("powTag-CustomerId:", q.Get("c")), fmt.Sprintf("powTag-EventId:", q.Get("e")), fmt.Sprintf("powTag-UserId:", Queueit)},

			"stats": map[string]interface{}{
				"duration":       "2050",
				"tries":          "1",
				"userAgent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				"screen":         "1920 x 1080",
				"browser":        "Chrome",
				"browserVersion": "91.0.4472.124",
				"isMobile":       false,
				"os":             "Windows",
				"osVersion":      "10",
				"cookiesEnabled": true,
			},
			"parameters": map[string]interface{}{
				"input":     input,
				"zeroCount": zeroCountInt,
			},
		}

		byteProof, _ := json.Marshal(proof)

		proofToken := base64.StdEncoding.EncodeToString(byteProof)

		datachallenge := map[string]interface{}{
			"challengeType": "proofofwork",
			"customerId":    "footlocker",
			"eventId":       q.Get("e"),
			"sessionId":     proofToken,
			"version":       "5",
		}

		payloadchallenge, _ := json.Marshal(datachallenge)

		//verify proof of work
		reqVerify, _ := http.NewRequest("POST", "https://footlocker.queue-it.net/challengeapi/verify", bytes.NewReader(payloadchallenge))
		reqVerify.Header.Set("Host", "footlocker.queue-it.net") //
		reqVerify.Header.Set("Connection", "keep-alive")
		reqVerify.Header.Set("Content-Length", "1545")
		reqVerify.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		reqVerify.Header.Set("X-Requested-With", "XMLHttpRequest")
		reqVerify.Header.Set("sec-ch-ua-mobile", "?0")
		reqVerify.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
		reqVerify.Header.Set("Content-Type", "application/json")
		reqVerify.Header.Set("Origin", "https://footlocker.queue-it.net")
		reqVerify.Header.Set("sec-fetch-site", "same-origin")
		reqVerify.Header.Set("sec-fetch-mode", "cors")
		reqVerify.Header.Set("sec-fetch-dest", "empty")
		reqVerify.Header.Set("Referer", redirectURL.String())
		reqVerify.Header.Set("accept-encoding", "gzip, deflate, br")
		reqVerify.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
		reqVerify.Header.Set("Cookie", fmt.Sprintf("Queue-it=u=", Queueit))
		resp, err := session.Client.Do(reqcap)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {

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

		case 200:
			challengeTypeP := gjson.Get(string(body), "sessionInfo.challengeType").String()
			checksumP := gjson.Get(string(body), "sessionInfo.checksum").String()
			sessionIdP := gjson.Get(string(body), "sessionInfo.sessionId").String()
			scourceIpP := gjson.Get(string(body), "sessionInfo.scourceIp").String()
			timestampP := gjson.Get(string(body), "sessionInfo.timestamp").String()
			versionP := gjson.Get(string(body), "sessionInfo.version").String()

			dataEnqueue := map[string]interface{}{
				"challengeSessions": []interface{}{
					map[string]interface{}{
						"sessionIdC":     sessionIdC,
						"timestampC":     timestampC,
						"checksumC":      checksumC,
						"scourceIpC":     scourceIpC,
						"challengeTypeC": challengeTypeC,
						"versionC":       versionC,
					}, map[string]interface{}{
						"sessionIdP":     sessionIdP,
						"timestampP":     timestampP,
						"checksumP":      checksumP,
						"scourceIpP":     scourceIpP,
						"challengeTypeP": challengeTypeP,
						"versionP":       versionP,
					},
				},

				"layoutName":      "FL Layout v02",
				"customUrlParams": " ",
				"Referrer":        "https://www.footlocker.com/",
				"targetUrl":       fmt.Sprintf("https://www.footlocker.com/en/product/~/%s", session.SKU),
			}

			payload, _ := json.Marshal(dataEnqueue)

			req, _ := http.NewRequest("POST", fmt.Sprintf("https://footlocker.queue-it.net/spa-api/queue/footlocker/%s/enqueue?cid=en-US", q.Get("e")), bytes.NewReader(payload))
			req.Header.Set("Host", "footlocker.queue-it.net") //
			req.Header.Set("Connection", "keep-alive")
			req.Header.Set("Content-Length", "2499")
			req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
			req.Header.Set("sec-ch-ua-mobile", "?0")
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://footlocker.queue-it.net")
			req.Header.Set("sec-fetch-site", "same-origin")
			req.Header.Set("sec-fetch-mode", "cors")
			req.Header.Set("sec-fetch-dest", "empty")
			req.Header.Set("Referer", redirectURL.String())
			req.Header.Set("accept-encoding", "gzip, deflate, br")
			req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
			req.Header.Set("Cookie", fmt.Sprintf("Queue-it=u=", Queueit))
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
			case 200:

			status:
				queueId := gjson.Get(string(body), "queueId").String()
				payload := map[string]interface{}{
					"targetUrl":               fmt.Sprintf("https://www.footlocker.com/en/product/~/%s", session.SKU),
					"customUrlParams":         "",
					"layoutVersion":           164616744217,
					"layoutName":              "FL Layout v02",
					"isClientRedayToRedirect": true,
					"isBeforeOrIdle":          false,
				}

				payload1, _ := json.Marshal(payload)
				req, _ := http.NewRequest("POST", fmt.Sprintf("https://footlocker.queue-it.net/spa-api/queue/footlocker/%s/%s/status", q.Get("e"), queueId), bytes.NewReader(payload1))
				req.Header.Set("Host", "footlocker.queue-it.net")
				req.Header.Set("Connection", "keep-alive")
				req.Header.Set("Content-Length", "2499")
				req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
				req.Header.Set("X-Requested-With", "XMLHttpRequest")
				req.Header.Set("sec-ch-ua-mobile", "?0")
				req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "https://footlocker.queue-it.net")
				req.Header.Set("sec-fetch-site", "same-origin")
				req.Header.Set("sec-fetch-mode", "cors")
				req.Header.Set("sec-fetch-dest", "empty")
				req.Header.Set("Referer", redirectURL.String())
				req.Header.Set("accept-encoding", "gzip, deflate, br")
				req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
				req.Header.Set("Cookie", fmt.Sprintf("Queue-it=u=", Queueit))

				query := req.URL.Query()
				query.Add("cid", q.Get("cid"))
				query.Add("l", "FL Layout v02")
				query.Add("seid", session.UUID)
				query.Add("sets", strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10))

				req.URL.RawQuery = query.Encode()
				resp, err := session.Client.Do(req)
				if err != nil {
					return err
				}

				body, err := ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				redirect := gjson.Get(string(body), "redirectUrl").String()

				if redirect != "" {

					req, _ := http.NewRequest("Get", redirect, nil)
					req.Header.Set("Host", "footlocker.queue-it.net")
					req.Header.Set("Connection", "keep-alive")
					req.Header.Set("sec-ch-ua-mobile", "?0")
					req.Header.Set("Upgrade-Insecure-Requests", "1")
					req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36")
					req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
					req.Header.Set("Sec-Fetch-Site", "cross-site")
					req.Header.Set("Sec-Fetch-Mode", "navigate")
					req.Header.Set("sec-Fetch-User", "?1")
					req.Header.Set("Sec-Fetch-Dest", "document")
					req.Header.Set("Referer", "http://footlocker.queue-it.net/")
					req.Header.Set("accept-encoding", "gzip, deflate, br")
					req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
					req.Header.Set("X-Fl-Request-Id", session.UUID)
					req.Header.Set("X-Flapi-session-id", session.JSESSIONID)
					req.Header.Set("Cookie", "JSESSIONID="+session.JSESSIONID)
					resp, _ := session.Client.Do(req)

					defer resp.Body.Close()
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

					case 429:
						fmt.Println("429")
						return errors.New("429")

					case 403:
						fmt.Println("403")
						return errors.New("403")
					case 503:
						fmt.Println("503")
						return errors.New("503")
					}
					return errors.New("UNKNOWN")

				} else {
					goto status
				}
			}
			return errors.New("UNKNOWN")
		}
		return errors.New("UNKNOWN")
	}
	return errors.New("UNKNOWN")
}
return errors.New("UNKNOWN")