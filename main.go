package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	Core "Footsite/Core"

	"github.com/TwinProduction/go-color"
)

var (
	wg          sync.WaitGroup
	site        string
	siteSKU     string
	sku         string
	sizes       string
	profileName string
	randSize    string = "7.0,7.5,8.0,8.5,9.0,9.5,10.0,10.5,11.0,11.5"
)

func LoadConfi(file string) (string, string, string) {

	f, err := os.Open(file)

	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	cap := scanner.Text()
	scanner.Scan()
	anti := scanner.Text()
	scanner.Scan()
	delay := scanner.Text()
	return cap, anti, delay
}

// func init(){}
func main() {

	//proxy
	proxyStr, _ := ioutil.ReadFile("data\\proxy.txt")
	proxies, _ := Core.LoadProxies(string(proxyStr))
	//profiles
	profiles, _ := Core.LoadProfile("data\\profiles.csv")

	Cap, Anti, Delay := LoadConfi("data\\config.txt")

	Cap = Cap[11:]
	Anti = Anti[13:]
	delay, _ := strconv.Atoi(Delay[6:])
	var config int
	if Cap != "" && Anti != "" && delay != 0 {
		config = 3
	} else if (Cap != "" || Anti != "") && delay != 0 {
		config = 2
	} else {
		config = 1
	}

	tasks, _ := Core.LoadTasks("data\\tasks.csv")
	fmt.Print(color.Green)
	fmt.Printf("%v proxies have been loaded.\n", len(proxies))
	fmt.Printf("%v profiles have been loaded.\n", len(profiles))
	fmt.Printf("%v config have been loaded.\n", config)
	fmt.Printf("%v tasks have been loaded.\n", len(tasks))
	fmt.Print(color.Reset)
	time.Sleep(2 * time.Second)
	var a string
	fmt.Print(color.Cyan)
	fmt.Println("Press enter to Start")
	fmt.Print(color.Reset)
	fmt.Scanln(&a)

	for i, _ := range tasks {
		wg.Add(1)

		go func(I int) {
			site = tasks[I].SiteIput
			switch site {
			case "FOOTLOCKER":
				site = "footlocker"
				siteSKU = "FL"
			case "EASTBADY":
				site = "eastbay"
				siteSKU = "EB"
			case "CHAMPSSPORTS":
				site = "champssports"
				siteSKU = "CS"
			case "FOOTACTION":
				site = "footaction"
				siteSKU = "FA"
			case "KIDSFOOTLOCKER":
				site = "kidsfootlocker"
				siteSKU = "KFL"
			}

			sku = tasks[I].Sku
			sizes = tasks[I].Sizes

			profileName = tasks[I].ProfileName
			tasks, _ := strconv.Atoi(tasks[I].TaskNum)

			for i, _ := range profiles {
				if profiles[i].ProfileName == profileName {

					switch sizes {

					case "RANDOM":
						sizeRange := strings.Split(randSize, ",")
						I, _ := strconv.ParseFloat(sizeRange[rand.Intn(len(sizeRange))], 64)

						if I >= 10 {

							Core.CreateTasks(site, siteSKU, sku, fmt.Sprintf("%.1f", I), proxies, profiles[i], tasks, Cap, Anti, delay)
							time.Sleep(2 * time.Second)
						} else {
							size := "0" + fmt.Sprintf("%.1f", I)

							Core.CreateTasks(site, siteSKU, sku, size, proxies, profiles[i], tasks, Cap, Anti, delay)
							time.Sleep(2 * time.Second)
						}
					default:
						sizeRange := strings.Split(sizes, ",")
						I, _ := strconv.ParseFloat(sizeRange[rand.Intn(len(sizeRange))], 64)

						if I >= 10 {

							Core.CreateTasks(site, siteSKU, sku, fmt.Sprintf("%.1f", I), proxies, profiles[i], tasks, Cap, Anti, delay)
							time.Sleep(2 * time.Second)
						} else {
							size := "0" + fmt.Sprintf("%.1f", I)

							Core.CreateTasks(site, siteSKU, sku, size, proxies, profiles[i], tasks, Cap, Anti, delay)
							time.Sleep(2 * time.Second)
						}
					}

				} else {
					rand.Seed(time.Now().UnixNano())
					profile := profiles[rand.Intn(len(profiles))]

					switch sizes {

					case "RANDOM":
						sizeRange := strings.Split(randSize, ",")
						I, _ := strconv.ParseFloat(sizeRange[rand.Intn(len(sizeRange))], 64)

						if I >= 10 {

							Core.CreateTasks(site, siteSKU, sku, fmt.Sprintf("%.1f", I), proxies, profile, tasks, Cap, Anti, delay)
						} else {
							size := "0" + fmt.Sprintf("%.1f", I)

							Core.CreateTasks(site, siteSKU, sku, size, proxies, profile, tasks, Cap, Anti, delay)
						}
					default:
						sizeRange := strings.Split(sizes, ",")
						I, _ := strconv.ParseFloat(sizeRange[rand.Intn(len(sizeRange))], 64)

						if I >= 10 {

							Core.CreateTasks(site, siteSKU, sku, fmt.Sprintf("%.1f", I), proxies, profile, tasks, Cap, Anti, delay)
						} else {
							size := "0" + fmt.Sprintf("%.1f", I)

							Core.CreateTasks(site, siteSKU, sku, size, proxies, profile, tasks, Cap, Anti, delay)
						}
					}

				}
			}
			wg.Done()

		}(i)

	}
	wg.Wait()

}
