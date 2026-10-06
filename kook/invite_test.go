package kook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInviteesStatusFilter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status *int
		value  string
	}{
		{"default", nil, ""},
		{"retained", testPtr(0), "0"},
		{"left", testPtr(254), "254"},
		{"all", testPtr(-1), "-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := map[string]string{"page": "1", "page_size": "20"}
				if tc.status != nil {
					query["status"] = tc.value
				}
				require.Equal(t, queryValues(query), r.URL.Query())
				_, _ = io.WriteString(w, `{"code":0,"data":{"items":[],"meta":{},"sort":[],"count":0,"keep_count":0,"loss_count":0}}`)
			}))
			defer server.Close()
			client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
			defer func() { _ = client.Close() }()
			result, err := client.Invite.GetInvitees(context.Background(), InviteeListParams{Status: tc.status, Page: 1, PageSize: 20})
			require.NoError(t, err)
			require.Empty(t, result.Items)
		})
	}
}

func TestGetInviteesRequiresPagination(t *testing.T) {
	client := NewClient("test-token", WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	for _, params := range []InviteeListParams{
		{}, {Page: 1}, {PageSize: 20}, {Page: -1, PageSize: 20}, {Page: 1, PageSize: -1},
	} {
		result, err := client.Invite.GetInvitees(context.Background(), params)
		require.ErrorContains(t, err, "页码和每页数量必须大于0")
		require.Nil(t, result)
	}
}

func TestGetInviteesErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"api", `{"code":40000,"message":"invite unavailable","data":{}}`},
		{"decode", `{"code":0,"data":{"items":"invalid"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
			defer func() { _ = client.Close() }()
			result, err := client.Invite.GetInvitees(context.Background(), InviteeListParams{Page: 1, PageSize: 20})
			require.Error(t, err)
			require.Nil(t, result)
			if tc.name == "api" {
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
			} else {
				require.ErrorContains(t, err, "解析被邀请用户列表失败")
			}
		})
	}
}
