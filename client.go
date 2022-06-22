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
	"net/http"
	"time"

	"github.com/imroc/req/v3"
)

var (
	browserClient, browserDownloadClient, cloudAPIClient, cloudFileClientTimeout2Min, cloudFileClientTimeout15s *req.Client

	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.71 Safari/537.36"
	siyuanUserAgent  = "SiYuan/0.0.0"
)

func InitHttpClient(siyuanUA string) {
	siyuanUserAgent = siyuanUA
}

func NewBrowserRequest(proxyURL string) (ret *req.Request) {
	if nil == browserClient {
		browserClient = req.C().
			SetUserAgent(browserUserAgent).
			SetTimeout(7 * time.Second).
			DisableInsecureSkipVerify()
	}
	if "" != proxyURL {
		browserClient.SetProxyURL(proxyURL)
	}
	ret = browserClient.R()
	ret.SetRetryCount(1).SetRetryFixedInterval(3 * time.Second)
	return
}

func NewBrowserDownloadRequest(proxyURL string) *req.Request {
	if nil == browserDownloadClient {
		browserDownloadClient = req.C().
			SetUserAgent(browserUserAgent).
			SetTimeout(2 * time.Minute).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify()
	}
	if "" != proxyURL {
		browserDownloadClient.SetProxyURL(proxyURL)
	}
	return browserDownloadClient.R()
}

func NewCloudRequest(proxyURL string) *req.Request {
	if nil == cloudAPIClient {
		cloudAPIClient = req.C().
			SetUserAgent(siyuanUserAgent).
			SetTimeout(7 * time.Second).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify()
	}
	if "" != proxyURL {
		cloudAPIClient.SetProxyURL(proxyURL)
	}
	return cloudAPIClient.R()
}

func NewCloudFileRequest2m(proxyURL string) *req.Request {
	if nil == cloudFileClientTimeout2Min {
		cloudFileClientTimeout2Min = req.C().
			SetUserAgent(siyuanUserAgent).
			SetTimeout(2 * time.Minute).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify()
		setTransport(cloudFileClientTimeout2Min.GetClient())
	}
	if "" != proxyURL {
		cloudFileClientTimeout2Min.SetProxyURL(proxyURL)
	}
	return cloudFileClientTimeout2Min.R()
}

func NewCloudFileRequest15s(proxyURL string) *req.Request {
	if nil == cloudFileClientTimeout15s {
		cloudFileClientTimeout15s = req.C().
			SetUserAgent(siyuanUserAgent).
			SetTimeout(15 * time.Second).
			SetCommonRetryCount(1).
			SetCommonRetryFixedInterval(3 * time.Second).
			SetCommonRetryCondition(retryCondition).
			DisableInsecureSkipVerify()
		setTransport(cloudFileClientTimeout15s.GetClient())
	}
	if "" != proxyURL {
		cloudFileClientTimeout15s.SetProxyURL(proxyURL)
	}
	return cloudFileClientTimeout15s.R()
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

func setTransport(client *http.Client) {
	// 改进同步下载数据稳定性 https://github.com/siyuan-note/siyuan/issues/4994
	transport := client.Transport.(*req.Transport)
	transport.MaxIdleConns = 10
	transport.MaxIdleConnsPerHost = 2
	transport.MaxConnsPerHost = 2
}
