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
	"errors"
	"net/http"
	"testing"

	"github.com/imroc/req/v3"
)

// TestRetryCondition 直接覆盖未导出的重试判定逻辑：发生错误、响应为空、
// 底层 http.Response 为空或状态码 503 时应重试；其余正常响应不应重试。
func TestRetryCondition(t *testing.T) {
	tests := []struct {
		name string
		resp *req.Response
		err  error
		want bool
	}{
		{
			name: "发生错误时需要重试",
			resp: &req.Response{Response: &http.Response{StatusCode: http.StatusOK}},
			err:  errors.New("connection reset"),
			want: true,
		},
		{
			name: "响应为空时需要重试",
			resp: nil,
			err:  nil,
			want: true,
		},
		{
			name: "底层 http.Response 为空时需要重试",
			resp: &req.Response{Response: nil},
			err:  nil,
			want: true,
		},
		{
			name: "状态码 503 时需要重试",
			resp: &req.Response{Response: &http.Response{StatusCode: http.StatusServiceUnavailable}},
			err:  nil,
			want: true,
		},
		{
			name: "状态码 200 时不需要重试",
			resp: &req.Response{Response: &http.Response{StatusCode: http.StatusOK}},
			err:  nil,
			want: false,
		},
		{
			name: "状态码 404 时不需要重试",
			resp: &req.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryCondition(tt.resp, tt.err); got != tt.want {
				t.Fatalf("retryCondition() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

// TestSetUserAgent 验证 SetUserAgent 会更新包级 siyuanUserAgent 变量，
// 测试结束后还原为默认值，避免污染其它测试用例。
func TestSetUserAgent(t *testing.T) {
	original := siyuanUserAgent
	defer func() { siyuanUserAgent = original }()

	const customUA = "SiYuan/9.9.9-test"
	SetUserAgent(customUA)

	if siyuanUserAgent != customUA {
		t.Fatalf("SetUserAgent 后 siyuanUserAgent = %q, 期望 %q", siyuanUserAgent, customUA)
	}
}
