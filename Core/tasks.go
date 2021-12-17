package core

import (
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TwinProduction/go-color"
	"github.com/gocarina/gocsv"
	"github.com/mattia-git/anticaptcha"
	"github.com/mattia-git/go-capmonster"
)

var (
	wg sync.WaitGroup
)

type Tasks struct {
	SiteIput    string `csv:"STORE"`
	Sku         string `csv:"SKU"`
	Sizes       string `csv:"SIZE"`
	TaskNum     string `csv:"TASKSNUM"`
	ProfileName string `csv:"PROFILES_NAME"`
}

var (
	AddCart        int
	CheckOUT       int
	CheckOUTFailed int
)

func LoadTasks(filePath string) ([]Tasks, error) {
	f, _ := os.Open(filePath)

	defer f.Close()

	tasks := []Tasks{}

	err := gocsv.UnmarshalFile(f, &tasks)

	return tasks, err

}

func CreateTasks(site string, siteSKU string, sku string, size string, proxies []Proxy, profile Profile, tasks int, Cap string, Anti string, delay int) {

	logger := log.New(os.Stdout, "", 0)

	for i := 0; i < tasks; i++ {
		wg.Add(1)
		go func() {

			s := Session{}
			rand.Seed(time.Now().UnixNano())
			proxy := proxies[rand.Intn(len(proxies))]
			var c = &capmonster.Client{APIKey: Cap}
			var a = &anticaptcha.Client{APIKey: Anti}
			_ = a
			//
			// profile := profiles[rand.Intn(len(profiles))]
			s.InitSession(site, siteSKU, sku, size, proxy.ConString(), profile)

			for {
				logger.Printf(" [%s] [%s, sku=%s] Genereating Session\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU)
				if s.GenerateSession() == nil {
					break
				}

				time.Sleep(time.Duration(delay/1000) * time.Second)
				rand.Seed(time.Now().UnixNano())

				s.ProxyURL, _ = url.Parse(proxies[rand.Intn(len(proxies))].ConString())
				logger.Printf(" [%s] [%s, sku=%s] Rotating Proxies\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU)
			}

			for {
				//fmt.Println("prepare to get Item")
				err := s.GetSizeID(c)
				if err == nil {
					break
				}
				if err.Error() == "DONE" {
					logger.Printf(" [%s] [%s, sku=%s, size=%s] Captcha Token Received\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				}
				if err.Error() == "FATAL" {
					wg.Done()
					return
				}

				time.Sleep(1 * time.Second)
			}

			for {

				logger.Printf("%s [%s] [%s, sku=%s, size=%s] Adding To Cart %s\n", color.Yellow, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)
				err := s.AddToCart(c)
				if err == nil {
					break
				}
				if err.Error() == "DONE" {
					logger.Printf("%s [%s] [%s, sku=%s, size=%s] Captcha Token Received%s\n", color.Yellow, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)

				}
				if err.Error() == "FATAL" {
					logger.Printf("%s [%s] [%s, sku=%s, size=%s] Captcha Solved Unsuccessfully%s\n", color.Red, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)

					//wg.Done()
					//return
				}
				time.Sleep(time.Duration(delay/1000) * time.Second)
			}

			logger.Printf("%s [%s] [%s, sku=%s, size=%s] ATC Success %s\n", color.Green, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)
			AddCart = AddCart + 1
			for {
				logger.Printf(" [%s] [%s, sku=%s, size=%s] Submitting Email\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				if s.SubmitEmail() == nil {
					break
				}
				time.Sleep(2 * time.Second)
			}

			for {
				logger.Printf(" [%s] [%s, sku=%s, size=%s] Submitting Shipping\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				if s.SubmitShipping() == nil {
					break
				}
				time.Sleep(1 * time.Second)
			}

			for {
				logger.Printf(" [%s] [%s, sku=%s, size=%s] Submitting Billing\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				if s.SubmitBilling() == nil {
					break
				}
				time.Sleep(1 * time.Second)
			}

			for {
				logger.Printf(" [%s] [%s, sku=%s, size=%s] Submitting Person\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				if s.PickPerson() == nil {
					break
				}
				time.Sleep(1 * time.Second)
			}

			for i := 0; i < 1; i++ {
				logger.Printf(" [%s] [%s, sku=%s, size=%s] Submitting Order\n", time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size)
				if s.SubmitOrder() == nil {
					logger.Printf("%s [%s] [%s, sku=%s, size=%s] Check Email%s\n", color.Green, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)
					CheckOUT = CheckOUT + 1
					go SendSuccessWebhook(strings.Title(s.Site), s.SKU, s.Size, "https://discord.com/api/webhooks/856412803178692649/mfX02_Wp0cEUpmh8PC-1pg9q9grL9OhXCWjWoYdqB0Fh-BHGmaTXxaeswZox44E2T11W")
					wg.Done()
					return
				} else {
					logger.Printf("%s [%s] [%s, sku=%s, size=%s] Payment Decline%s\n", color.Red, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, s.Size, color.Reset)

				}
			}
			wg.Add(1)
			go SendDeclineWebhook(strings.Title(s.Site), s.SKU, s.Size, "https://discord.com/api/webhooks/856412803178692649/mfX02_Wp0cEUpmh8PC-1pg9q9grL9OhXCWjWoYdqB0Fh-BHGmaTXxaeswZox44E2T11W")
			wg.Wait()
			CheckOUTFailed = CheckOUTFailed + 1
			logger.Printf("%s [%s] [%s, sku=%s, AddToCart=%d, CheckOut=%d, CheckOUTFailed=%d] Check Sum%s\n", color.Cyan, time.Now().Format("2006-01-02 15:04:05"), s.Site, s.SKU, AddCart, CheckOUT, CheckOUTFailed, color.Reset)
			wg.Done()

		}()

	}
	wg.Wait()

}
