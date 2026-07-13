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
	browserClient, cloudClientTimeout30s, cloudFileClientTimeout2Min *req.Client

	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
	siyuanUserAgent  = "SiYuan/0.0.0"
)

func CloseIdleConnections() {
	if nil != browserClient {
		browserClient.GetClient().CloseIdleConnections()
	}
	if nil != cloudClientTimeout30s {
		cloudClientTimeout30s.GetClient().CloseIdleConnections()
	}
	if nil != cloudFileClientTimeout2Min {
		cloudFileClientTimeout2Min.GetClient().CloseIdleConnections()
	}
}

func GetCloudFileClient2Min() *http.Client {
	if nil == cloudFileClientTimeout2Min {
		newCloudFileClient2m()
	}
	return cloudFileClientTimeout2Min.GetClient()
}

func SetUserAgent(siyuanUA string) {
	siyuanUserAgent = siyuanUA
}

// UserAgentTransport 在出站请求上注入 SiYuan User-Agent，统一内核出网身份标识。
// 使用 Set 覆盖，会替换请求上已有的 UA（如 aws SDK 构造的含架构、Go 版本等的长 UA 串），
// 便于第三方服务端按 SiYuan/ 前缀稳定识别并加白名单。
// Base 为底座 transport，负责实际的连接、代理、TLS 等配置。
type UserAgentTransport struct {
	Base http.RoundTripper
}

func (t *UserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", siyuanUserAgent)
	return base.RoundTrip(clone)
}

// NewUserAgentRoundTripper 返回带 SiYuan UA 注入的 RoundTripper。
// base 为 nil 时回退到 http.DefaultTransport。适用于需要在 transport 层叠加自定义逻辑的场景（如 MCP 自定义 header）。
func NewUserAgentRoundTripper(base http.RoundTripper) http.RoundTripper {
	return &UserAgentTransport{Base: base}
}

// NewUserAgentClient 返回带 SiYuan UA 注入的 *http.Client。
// base 为 nil 时用 NewTransport(false) 作底座。调用方可自行设置返回 client 的 Timeout。
func NewUserAgentClient(base http.RoundTripper) *http.Client {
	if base == nil {
		base = NewTransport(false)
	}
	return &http.Client{Transport: &UserAgentTransport{Base: base}}
}

func NewBrowserRequest() (ret *req.Request) {
	if nil == browserClient {
		browserClient = req.C().
			SetUserAgent(browserUserAgent).
			SetTimeout(30 * time.Second).
			DisableInsecureSkipVerify().
			SetProxy(ProxyFromEnvironment)
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
		EnableForceHTTP1(). // 强制使用 HTTP/1.1，避免有些服务器并发请求时报错 https://github.com/siyuan-note/siyuan/issues/6948
		SetCommonHeader("Cache-Control", "no-cache, no-store, must-revalidate").
		SetCommonHeader("Pragma", "no-cache").
		SetCommonHeader("Expires", "0").
		SetUserAgent(siyuanUserAgent).
		SetTimeout(2 * time.Minute).
		SetCommonRetryCount(1).
		SetCommonRetryFixedInterval(3 * time.Second).
		SetCommonRetryCondition(retryCondition).
		DisableInsecureSkipVerify().
		SetProxy(ProxyFromEnvironment)
}

func NewCloudRequest30s() *req.Request {
	if nil == cloudClientTimeout30s {
		cloudClientTimeout30s = req.C().
			SetUserAgent(siyuanUserAgent).
			SetTimeout(30 * time.Second).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify().
			SetProxy(ProxyFromEnvironment)
	}
	return cloudClientTimeout30s.R()
}

func retryCondition(resp *req.Response, err error) bool {
	if nil != err {
		return true
	}
	if nil == resp || nil == resp.Response {
		return true
	}
	if 503 == resp.StatusCode { // 返回 503 需要重试
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
