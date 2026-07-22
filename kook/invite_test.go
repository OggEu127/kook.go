package kook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInviteesContractAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v3/invite/invitees", r.URL.Path)
		require.Equal(t, "code", r.URL.Query().Get("id"))
		require.Equal(t, "https://kook.vip/code", r.URL.Query().Get("invite_url"))
		require.Equal(t, "guild", r.URL.Query().Get("guild_id"))
		require.Equal(t, "254", r.URL.Query().Get("status"))
		require.Equal(t, "2026-06-01 12:00:00", r.URL.Query().Get("start_time"))
		require.Equal(t, "2026-07-01 12:00:00", r.URL.Query().Get("end_time"))
		require.Equal(t, "1", r.URL.Query().Get("page"))
		require.Equal(t, "20", r.URL.Query().Get("page_size"))
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"items":[{"status":254,"joined_time":1773643290000,"active_time":1773643289899,"show_name":"user#0001"}],"meta":{"page":1,"page_total":1,"page_size":20,"total":1},"sort":{},"count":3,"keep_count":2,"loss_count":1}}`))
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	result, err := client.Invite.GetInvitees(context.Background(), InviteeListParams{
		ID: "code", InviteURL: "https://kook.vip/code", GuildID: "guild", Status: testPtr(InviteeStatusLeft),
		StartTime: "2026-06-01 12:00:00", EndTime: "2026-07-01 12:00:00", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.Count)
	require.Equal(t, 2, result.KeepCount)
	require.Equal(t, 1, result.LossCount)
	require.Equal(t, "user#0001", result.Items[0].ShowName)
	require.Equal(t, int64(1773643290000), result.Items[0].JoinedTime)
}

func TestGetInviteesValidation(t *testing.T) {
	client := NewClient("token", WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	_, err := client.Invite.GetInvitees(context.Background(), InviteeListParams{PageSize: 20})
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "page", validationErr.Field)

	invalidStatus := 1
	_, err = client.Invite.GetInvitees(context.Background(), InviteeListParams{Page: 1, PageSize: 20, Status: &invalidStatus})
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "status", validationErr.Field)
}
