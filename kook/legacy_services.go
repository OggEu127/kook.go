package kook

import (
	"context"
	"fmt"
)

func unsupportedEndpoint(endpoint string) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedEndpoint, endpoint)
}

// 下列服务和类型仅保留 v1.1.1 的源码兼容性。对应端点未出现在当前官方接口清单中。
type AdminService struct{ client *Client }

func (*AdminService) GetAuditLog(context.Context, string, string, string, int, int, int) (*AuditLogResponse, error) {
	return nil, unsupportedEndpoint("guild/audit-log")
}
func (*AdminService) BanUser(context.Context, string, string, string, int) error {
	return unsupportedEndpoint("guild/ban")
}
func (*AdminService) UnbanUser(context.Context, string, string) error {
	return unsupportedEndpoint("guild/unban")
}
func (*AdminService) GetBannedUsers(context.Context, string, int, int) (*BannedUsersResponse, error) {
	return nil, unsupportedEndpoint("guild/ban-list")
}

type AuditLogEntry struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	User       User                   `json:"user"`
	TargetID   string                 `json:"target_id"`
	ActionType int                    `json:"action_type"`
	Reason     string                 `json:"reason"`
	Options    map[string]interface{} `json:"options"`
	CreatedAt  int64                  `json:"created_at"`
}
type AuditLogResponse struct {
	Items []AuditLogEntry `json:"items"`
	Meta  PaginationMeta  `json:"meta"`
	Sort  map[string]int  `json:"sort"`
}
type BannedUser struct {
	User     User   `json:"user"`
	Reason   string `json:"reason"`
	BannedAt int64  `json:"banned_at"`
	BannedBy string `json:"banned_by"`
}
type BannedUsersResponse struct {
	Items []BannedUser   `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

const (
	AuditLogActionGuildUpdate      = 1
	AuditLogActionChannelCreate    = 10
	AuditLogActionChannelUpdate    = 11
	AuditLogActionChannelDelete    = 12
	AuditLogActionRoleCreate       = 30
	AuditLogActionRoleUpdate       = 31
	AuditLogActionRoleDelete       = 32
	AuditLogActionMemberKick       = 20
	AuditLogActionMemberBan        = 22
	AuditLogActionMemberUnban      = 23
	AuditLogActionMemberUpdate     = 24
	AuditLogActionMemberRoleUpdate = 25
	AuditLogActionMessageDelete    = 72
)

type BoostService struct{ client *Client }

func (*BoostService) GetUnusedBoostNum(context.Context) (*UnusedBoostInfo, error) {
	return nil, unsupportedEndpoint("guild-boost/get-unused-boost-num")
}
func (*BoostService) UseBoost(context.Context, string, int) error {
	return unsupportedEndpoint("boost/use")
}
func (*BoostService) GetGuildBoosts(context.Context, string, int, int) (*GuildBoostListResponse, error) {
	return nil, unsupportedEndpoint("guild-boost/list")
}
func (*BoostService) CancelBoost(context.Context, string, string) error {
	return unsupportedEndpoint("boost/cancel")
}

type UnusedBoostInfo struct {
	UnusedBoostNum int `json:"unused_boost_num"`
}
type GuildBoost struct {
	ID        string `json:"id"`
	GuildID   string `json:"guild_id"`
	UserID    string `json:"user_id"`
	User      User   `json:"user"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Level     int    `json:"level"`
	Status    int    `json:"status"`
}
type GuildBoostListResponse struct {
	Items []GuildBoost   `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

type CouponService struct{ client *Client }

func (*CouponService) ExchangeCoupon(context.Context, string) (*CouponExchangeResult, error) {
	return nil, unsupportedEndpoint("coupon/exchange")
}
func (*CouponService) GetCoupons(context.Context, int, int) (*CouponListResponse, error) {
	return nil, unsupportedEndpoint("coupon/list")
}
func (*CouponService) UseCoupon(context.Context, string, string) error {
	return unsupportedEndpoint("coupon/use")
}

type Coupon struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        int    `json:"type"`
	Value       int    `json:"value"`
	MinAmount   int    `json:"min_amount"`
	ExpiredAt   int64  `json:"expired_at"`
	Used        bool   `json:"used"`
	UsedAt      int64  `json:"used_at"`
	CreatedAt   int64  `json:"created_at"`
}
type CouponExchangeResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Coupon  Coupon `json:"coupon,omitempty"`
	Items   []Item `json:"items,omitempty"`
}
type CouponListResponse struct {
	Items []Coupon       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

const (
	CouponTypeDiscount  = 1
	CouponTypeReduction = 2
)

type ItemService struct{ client *Client }

func (*ItemService) GetItemList(context.Context, string) (*ItemListResponse, error) {
	return nil, unsupportedEndpoint("item/list")
}
func (*ItemService) GetBag(context.Context) ([]BagItem, error) {
	return nil, unsupportedEndpoint("item/bag")
}
func (*ItemService) UseItem(context.Context, int) error { return unsupportedEndpoint("item/using") }
func (*ItemService) CancelUseItem(context.Context, int) error {
	return unsupportedEndpoint("item/cancel-use")
}
func (*ItemService) DeleteItems(context.Context, []int) error {
	return unsupportedEndpoint("item/delete")
}

type Item struct {
	ID            string `json:"id"`
	Status        int    `json:"status"`
	Type          int    `json:"type"`
	Name          string `json:"name"`
	Price         int    `json:"price"`
	OriginPrice   int    `json:"origin_price"`
	ServiceTime   int    `json:"service_time"`
	DiscountLabel string `json:"discount_label"`
	IAPCode       string `json:"iap_code"`
}
type BagItem struct {
	UserItemID int    `json:"user_item_id"`
	ItemID     string `json:"item_id"`
	Item       Item   `json:"item"`
	Count      int    `json:"count"`
	ExpiredAt  int64  `json:"expired_at"`
	Using      bool   `json:"using"`
}
type ItemListResponse struct {
	Items []Item `json:"items"`
}

const (
	ItemCategoryAll        = "all"
	ItemCategoryTimeLimit  = "time_limit"
	ItemCategoryDecoration = "decoration"
	ItemCategoryAction     = "action"
)

type LiveService struct{ client *Client }

func (*LiveService) StartLive(context.Context, string, string) (*LiveInfo, error) {
	return nil, unsupportedEndpoint("live/start")
}
func (*LiveService) StopLive(context.Context, string) error { return unsupportedEndpoint("live/stop") }
func (*LiveService) GetLiveInfo(context.Context, string) (*LiveInfo, error) {
	return nil, unsupportedEndpoint("live/info")
}

type LiveInfo struct {
	ChannelID   string `json:"channel_id"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	ViewerCount int    `json:"viewer_count"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	StreamURL   string `json:"stream_url"`
	PlayURL     string `json:"play_url"`
}

type OrderService struct{ client *Client }

func (*OrderService) CreateOrder(context.Context, CreateOrderParams) (*Order, error) {
	return nil, unsupportedEndpoint("order/create")
}
func (*OrderService) GetOrderStatus(context.Context, string) (*Order, error) {
	return nil, unsupportedEndpoint("order/status")
}
func (*OrderService) GetOrders(context.Context, int, int) (*OrderListResponse, error) {
	return nil, unsupportedEndpoint("order/list")
}

type CreateOrderParams struct {
	Products   []OrderProduct `json:"products"`
	Platform   int            `json:"platform"`
	RequestPay bool           `json:"request_pay"`
}
type OrderProduct struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}
type Order struct {
	ID               string    `json:"id"`
	Status           int       `json:"status"`
	UserID           string    `json:"user_id"`
	TotalFee         int       `json:"total_fee"`
	PayFee           int       `json:"pay_fee"`
	Paid             bool      `json:"paid"`
	PayTime          int64     `json:"pay_time"`
	CreateTime       int64     `json:"create_time"`
	Products         []Product `json:"products"`
	UsageInfo        string    `json:"usage_info"`
	ItemEntitiesDesc string    `json:"item_entities_desc"`
	PayData          *PayData  `json:"paydata,omitempty"`
}
type Product struct {
	ID         int         `json:"id"`
	ItemID     int         `json:"item_id"`
	Item       ProductItem `json:"item"`
	Total      int         `json:"total"`
	ExpireTime int64       `json:"expire_time"`
}
type ProductItem struct {
	ID              int                  `json:"id"`
	Name            string               `json:"name"`
	Desc            string               `json:"desc"`
	CD              int                  `json:"cd"`
	Categories      []string             `json:"categories"`
	Label           int                  `json:"label"`
	LabelName       string               `json:"label_name"`
	Quality         int                  `json:"quality"`
	Icon            string               `json:"icon"`
	IconThumb       string               `json:"icon_thumb"`
	IconExpired     string               `json:"icon_expired"`
	QualityResource QualityResource      `json:"quality_resource"`
	Resources       ProductItemResources `json:"resources"`
	Position        string               `json:"position"`
}
type QualityResource struct {
	Color string `json:"color"`
	Small string `json:"small"`
	Big   string `json:"big"`
}
type ProductItemResources struct {
	GIF            string `json:"gif"`
	Height         int    `json:"height"`
	PAG            string `json:"pag"`
	Percent        int    `json:"percent"`
	Preview        string `json:"preview"`
	PreviewExpired string `json:"preview_expired"`
	Time           int    `json:"time"`
	Type           string `json:"type"`
	WEBP           string `json:"webp"`
	Width          int    `json:"width"`
}
type PayData struct {
	ID          string `json:"id"`
	PayFee      string `json:"pay_fee"`
	QRCode      string `json:"qr_code"`
	QRCodeURL   string `json:"qr_code_url"`
	ExpiredTime int64  `json:"expired_time"`
	MobilePay   string `json:"mobile_pay"`
}
type OrderListResponse struct {
	Items []Order        `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

type RegionService struct{ client *Client }

func (*RegionService) GetRegionList(context.Context) ([]Region, error) {
	return nil, unsupportedEndpoint("guild/regions")
}

type Region struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Crowding int    `json:"crowding"`
}

type SecurityService struct{ client *Client }

func (*SecurityService) GetSecuritySettings(context.Context, string) (*SecuritySettings, error) {
	return nil, unsupportedEndpoint("guild-security/settings")
}
func (*SecurityService) UpdateSecuritySetting(context.Context, string, string, bool) error {
	return unsupportedEndpoint("guild-security/update")
}
func (*SecurityService) GetVerificationLevel(context.Context, string) (*VerificationLevel, error) {
	return nil, unsupportedEndpoint("guild/verification-level")
}
func (*SecurityService) UpdateVerificationLevel(context.Context, string, int) error {
	return unsupportedEndpoint("guild/verification-level")
}

type SecuritySettings struct {
	GuildID  string         `json:"guild_id"`
	Settings []SecurityRule `json:"settings"`
}
type SecurityRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Type        int    `json:"type"`
}
type VerificationLevel struct {
	GuildID string `json:"guild_id"`
	Level   int    `json:"level"`
}

const (
	VerificationLevelNone     = 0
	VerificationLevelLow      = 1
	VerificationLevelMedium   = 2
	VerificationLevelHigh     = 3
	VerificationLevelVeryHigh = 4
)

// OAuthService 是 v1 Client.OAuth 的兼容适配器；该端点仍在官方接口清单中。
type OAuthService struct{ client *Client }

func (s *OAuthService) GetOAuthToken(ctx context.Context, grantType, clientID, clientSecret, code, redirectURI string) (*OAuthTokenResponse, error) {
	oauth := NewOAuthClient(WithOAuthHTTPClient(s.client.httpClient), WithOAuthBaseURL(s.client.baseURL))
	return oauth.ExchangeToken(ctx, OAuthTokenParams{GrantType: grantType, ClientID: clientID, ClientSecret: clientSecret, Code: code, RedirectURI: redirectURI})
}
