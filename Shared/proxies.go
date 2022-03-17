package Shared

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"h12.io/socks"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	HTTP = iota
	SOCKS4
	SOCKS4A
	SOCKS5
)

type ProxyManager struct {
	ProxyList     []*Proxy
	ProxyType     int
	ProxyAuthUser string
	ProxyAuthPass string
}

type Proxy struct {
	Address string
	InUse   bool
	Banned  bool
}

func (pm *ProxyManager) GetRandomProxy() *Proxy {
	if len(pm.ProxyList) == 0 {
		return nil
	} else if len(pm.ProxyList) == 1 {
		return pm.ProxyList[0]
	}

	var p *Proxy
	for {
		p = pm.ProxyList[rand.Intn(len(pm.ProxyList)-1)]
		if !p.InUse && !p.Banned {
			break
		}
		if pm.GetLivingCount() == 0 {
			for _, p := range pm.ProxyList {
				p.Banned = false
			}
		}
	}

	p.InUse = true
	return p
}

func (pm *ProxyManager) GetLivingCount() int {
	i := 0
	for _, p := range pm.ProxyList {
		if p.Banned == false {
			i++
		}
	}
	return i
}

func (pm *ProxyManager) LoadFromFile(filename string, proxyType int) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	pm.ProxyList = []*Proxy{}
	pm.ProxyType = proxyType
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Count(line, ":") == 3 {
			a := strings.Split(line, ":")
			pm.ProxyList = append(pm.ProxyList, &Proxy{Address: strings.Join(a[:2], ":"), InUse: false})
			pm.ProxyAuthUser = a[2]
			pm.ProxyAuthPass = a[3]
		} else {
			pm.ProxyList = append(pm.ProxyList, &Proxy{Address: line, InUse: false})
		}
	}
	return len(pm.ProxyList), nil
}

func (pm *ProxyManager) GetRandomProxyTransport() (*http.Transport, *Proxy, error) {
	customTransport := &http.Transport{}

	proxy := pm.GetRandomProxy()

	if proxy == nil {
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS11, MaxVersion: tls.VersionTLS11}
		customTransport.TLSClientConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return nil
		}

		customTransport.IdleConnTimeout = 10 * time.Second
		return customTransport, nil, nil
	}
	switch PManager.ProxyType {
	case HTTP:
		proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s", proxy.Address))
		customTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
		break
	case SOCKS4:
		dialSocksProxy := socks.Dial(fmt.Sprintf("socks4://%s?timeout=%ds", proxy.Address, 10))
		customTransport = &http.Transport{Dial: dialSocksProxy}
		break
	case SOCKS5:
		dialSocksProxy := socks.Dial(fmt.Sprintf("socks5://%s?timeout=%ds", proxy.Address, 10))
		customTransport = &http.Transport{Dial: dialSocksProxy}
		break
	}

	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS11, MaxVersion: tls.VersionTLS11}
	customTransport.TLSClientConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		return nil
	}

	customTransport.IdleConnTimeout = 10 * time.Second

	if len(PManager.ProxyAuthUser) > 0 {
		customTransport.ProxyConnectHeader = http.Header{}
		customTransport.ProxyConnectHeader.Add("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", PManager.ProxyAuthUser, PManager.ProxyAuthPass))))
	}

	return customTransport, proxy, nil
}
