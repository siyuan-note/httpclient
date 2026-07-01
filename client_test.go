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

package httpclient_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	httpclient "github.com/siyuan-note/httpclient"
)

// 浏览器客户端使用的 User-Agent，需与 client.go 中的 browserUserAgent 保持一致。
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

// TestGetCloudFileClient2Min 验证 2 分钟云文件客户端可被懒加载创建，
// 且底层 http.Client 的超时时间为 2 分钟。
func TestGetCloudFileClient2Min(t *testing.T) {
	client := httpclient.GetCloudFileClient2Min()
	if client == nil {
		t.Fatal("GetCloudFileClient2Min() 返回 nil")
	}
	if want, got := 2*time.Minute, client.Timeout; got != want {
		t.Fatalf("client.Timeout = %v, 期望 %v", got, want)
	}
}

// TestNewCloudFileRequest2m 验证 2 分钟云文件请求对象可被创建且非空。
func TestNewCloudFileRequest2m(t *testing.T) {
	if req := httpclient.NewCloudFileRequest2m(); req == nil {
		t.Fatal("NewCloudFileRequest2m() 返回 nil")
	}
}

// TestNewCloudRequest30s 验证 30 秒云请求对象可被创建且非空。
func TestNewCloudRequest30s(t *testing.T) {
	if req := httpclient.NewCloudRequest30s(); req == nil {
		t.Fatal("NewCloudRequest30s() 返回 nil")
	}
}

// TestNewBrowserRequest 验证浏览器请求对象可被创建且非空。
func TestNewBrowserRequest(t *testing.T) {
	if req := httpclient.NewBrowserRequest(); req == nil {
		t.Fatal("NewBrowserRequest() 返回 nil")
	}
}

// TestNewTransport 验证 NewTransport 返回的 *http.Transport 关键配置项符合预期，
// 且 InsecureSkipVerify 随入参变化。
func TestNewTransport(t *testing.T) {
	tr := httpclient.NewTransport(false)

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"MaxIdleConns", tr.MaxIdleConns, int(10)},
		{"MaxIdleConnsPerHost", tr.MaxIdleConnsPerHost, int(2)},
		{"MaxConnsPerHost", tr.MaxConnsPerHost, int(2)},
		{"IdleConnTimeout", tr.IdleConnTimeout, 90 * time.Second},
		{"TLSHandshakeTimeout", tr.TLSHandshakeTimeout, 7 * time.Second},
		{"ExpectContinueTimeout", tr.ExpectContinueTimeout, 1 * time.Second},
		{"ForceAttemptHTTP2", tr.ForceAttemptHTTP2, true},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, 期望 %v", c.name, c.got, c.want)
		}
	}
	if tr.Proxy == nil {
		t.Error("Proxy 函数为 nil, 期望使用 ProxyFromEnvironment")
	}

	// skipTlsVerify 参数应透传到 TLSClientConfig.InsecureSkipVerify。
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("skipTlsVerify=false 时 InsecureSkipVerify 应为 false")
	}
	trVerify := httpclient.NewTransport(true)
	if trVerify.TLSClientConfig == nil || !trVerify.TLSClientConfig.InsecureSkipVerify {
		t.Error("skipTlsVerify=true 时 InsecureSkipVerify 应为 true")
	}
}

// TestProxyFromEnvironment 验证 ProxyFromEnvironment 能根据环境变量返回代理地址，
// 并在未设置代理时返回 nil。
func TestProxyFromEnvironment(t *testing.T) {
	t.Run("根据环境变量返回代理", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "http://127.0.0.1:18080")
		t.Setenv("NO_PROXY", "")

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("构造请求失败: %v", err)
		}
		u, err := httpclient.ProxyFromEnvironment(req)
		if err != nil {
			t.Fatalf("ProxyFromEnvironment 返回错误: %v", err)
		}
		if u == nil {
			t.Fatal("期望返回代理 URL, 实际为 nil")
		}
		if want, got := "127.0.0.1:18080", u.Host; got != want {
			t.Fatalf("代理 Host = %q, 期望 %q", got, want)
		}
	})

	t.Run("未设置代理时返回 nil", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("NO_PROXY", "*")

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("构造请求失败: %v", err)
		}
		u, err := httpclient.ProxyFromEnvironment(req)
		if err != nil {
			t.Fatalf("ProxyFromEnvironment 返回错误: %v", err)
		}
		if u != nil {
			t.Fatalf("期望代理为 nil, 实际为 %v", u)
		}
	})
}

// TestCloseIdleConnections 验证在客户端尚未初始化以及已初始化两种情况下调用均不会 panic。
func TestCloseIdleConnections(t *testing.T) {
	// 先在尚未初始化任何客户端时调用，确保幂等安全。
	httpclient.CloseIdleConnections()

	// 触发各客户端的懒加载初始化。
	httpclient.NewBrowserRequest()
	httpclient.NewCloudRequest30s()
	httpclient.NewCloudFileRequest2m()
	httpclient.GetCloudFileClient2Min()

	// 初始化后再次调用，确保不 panic。
	httpclient.CloseIdleConnections()
}

// TestNewBrowserRequest_E2E 通过本地测试服务器验证浏览器请求会携带预期的 User-Agent，
// 并且能成功收到 200 响应。
func TestNewBrowserRequest_E2E(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := httpclient.NewBrowserRequest().Get(server.URL)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if resp.Response == nil {
		t.Fatal("底层 http.Response 为 nil")
	}
	defer resp.Response.Body.Close()

	if resp.Response.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 %d", resp.Response.StatusCode, http.StatusOK)
	}
	if receivedUA != browserUserAgent {
		t.Fatalf("服务器收到 User-Agent = %q, 期望 %q", receivedUA, browserUserAgent)
	}
}

// TestRetryCondition_E2E_503 通过本地测试服务器验证：当首次响应 503 时，
// 云端请求会触发一次重试并在第二次请求 200 时成功返回。
func TestRetryCondition_E2E_503(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&count, 1) == 1 {
			// 首次请求返回 503，应触发重试。
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// 第二次请求返回 200。
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 注意：重试固定间隔为 3 秒，本用例至少耗时约 3 秒。
	resp, err := httpclient.NewCloudRequest30s().Get(server.URL)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if resp.Response == nil {
		t.Fatal("底层 http.Response 为 nil")
	}
	defer resp.Response.Body.Close()

	if got := resp.Response.StatusCode; got != http.StatusOK {
		t.Fatalf("重试后状态码 = %d, 期望 %d", got, http.StatusOK)
	}
	if got := atomic.LoadInt32(&count); got != 2 {
		t.Fatalf("服务器被请求次数 = %d, 期望 2（首次 503 触发一次重试）", got)
	}
}
