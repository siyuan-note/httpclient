// HttpClient - HTTP client for SiYuan.
// Copyright (c) 2022-present, b3log.org
//
// HttpClient is licensed under Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//         http://license.coscl.org.cn/MulanPSL2
//
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
// EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
// MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
//
// See the Mulan PSL v2 for more details.

package httpclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/imroc/req/v3"
	"golang.org/x/net/http/httpproxy"
)

var (
	browserClient, cloudFileClientTimeout2Min, cloudClientTimeout30s *req.Client

	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
	siyuanUserAgent  = "SiYuan/0.0.0"
)

func GetCloudFileClient2Min() *http.Client {
	if nil == cloudFileClientTimeout2Min {
		newCloudFileClient2m()
	}
	return cloudFileClientTimeout2Min.GetClient()
}

func SetUserAgent(siyuanUA string) {
	siyuanUserAgent = siyuanUA
}

func NewBrowserRequest() (ret *req.Request) {
	if nil == browserClient {
		browserClient = req.C().
			SetUserAgent(browserUserAgent).
			SetTimeout(30 * time.Second).
			DisableInsecureSkipVerify()
		browserClient.GetClient().Transport = NewTransport(false)
	}
	ret = browserClient.R()
	ret.SetRetryCount(1).SetRetryFixedInterval(3 * time.Second)
	return
}

func NewCloudFileRequest2m() *req.Request {
	if nil == cloudFileClientTimeout2Min {
		newCloudFileClient2m()
	}
	return cloudFileClientTimeout2Min.R()
}

func newCloudFileClient2m() {
	cloudFileClientTimeout2Min = req.C().
		SetUserAgent(siyuanUserAgent).
		SetTimeout(2 * time.Minute).
		SetCommonRetryCount(1).
		SetCommonRetryFixedInterval(3 * time.Second).
		SetCommonRetryCondition(retryCondition).
		DisableInsecureSkipVerify()
	cloudFileClientTimeout2Min.GetClient().Transport = NewTransport(false)
}

func NewCloudRequest30s() *req.Request {
	if nil == cloudClientTimeout30s {
		cloudClientTimeout30s = req.C().
			SetUserAgent(siyuanUserAgent).
			SetTimeout(30 * time.Second).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify()
		cloudClientTimeout30s.GetClient().Transport = NewTransport(false)
	}
	return cloudClientTimeout30s.R()
}

func retryCondition(resp *req.Response, err error) bool {
	if nil != err {
		return true
	}
	if 503 == resp.StatusCode { // 负载均衡会返回 503，需要重试
		return true
	}
	return false
}

func NewTransport(skipTlsVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: ProxyFromEnvironment,
		DialContext: defaultTransportDialContext(&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		MaxConnsPerHost:       2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   7 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: skipTlsVerify}}
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

func ProxyFromEnvironment(req *http.Request) (*url.URL, error) {
	// 因为 http.ProxyFromEnvironment 为了优化性能所以会缓存结果
	// 这里需要每次都重新从环境变量获取，以便实现不重启就能切换代理
	return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
}
