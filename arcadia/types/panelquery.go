package types

import "fmt"

// PanelQuery is the request union accepted by POST /. Exactly one field is
// non-nil. All variants are struct variants.
type PanelQuery struct {
	Authorize                   *QAuthorize
	Hello                       *QHello
	BaseAnalytics               *QLoginTokenOnly
	GetUser                     *QGetUser
	BotQueue                    *QLoginTokenOnly
	ExecuteRpc                  *QExecuteRpc
	GetRpcMethods               *QGetRpcMethods
	GetRpcLogEntries            *QLoginTokenOnly
	SearchEntitys               *QSearchEntitys
	UploadCdnFileChunk          *QUploadCdnFileChunk
	ListCdnScopes               *QLoginTokenOnly
	GetMainCdnScope             *QLoginTokenOnly
	UpdateCdnAsset              *QUpdateCdnAsset
	UpdatePartners              *QUpdatePartners
	UpdateChangelog             *QUpdateChangelog
	UpdateBlog                  *QUpdateBlog
	UpdateStaffPositions        *QUpdateStaffPositions
	UpdateStaffMembers          *QUpdateStaffMembers
	UpdateStaffDisciplinaryType *QUpdateStaffDisciplinaryType
	UpdateVoteCreditTiers       *QUpdateVoteCreditTiers
	UpdateShopItems             *QUpdateShopItems
	UpdateShopItemBenefits      *QUpdateShopItemBenefits
	UpdateShopCoupons           *QUpdateShopCoupons
	UpdateBotWhitelist          *QUpdateBotWhitelist
	PopplioStaff                *QPopplioStaff
}

// QLoginTokenOnly is the shape shared by the variants that take nothing but a
// login token.
type QLoginTokenOnly struct {
	LoginToken string `json:"login_token"`
}

type QAuthorize struct {
	Version uint16          `json:"version"`
	Action  AuthorizeAction `json:"action"`
}

type QHello struct {
	LoginToken string `json:"login_token"`
	Version    uint16 `json:"version"`
}

type QGetUser struct {
	LoginToken string `json:"login_token"`
	UserID     string `json:"user_id"`
}

type QExecuteRpc struct {
	LoginToken string     `json:"login_token"`
	TargetType TargetType `json:"target_type"`
	Method     RPCMethod  `json:"method"`
}

type QGetRpcMethods struct {
	LoginToken string `json:"login_token"`
	Filtered   bool   `json:"filtered"`
}

type QSearchEntitys struct {
	LoginToken string     `json:"login_token"`
	TargetType TargetType `json:"target_type"`
	Query      string     `json:"query"`
}

type QUploadCdnFileChunk struct {
	LoginToken string `json:"login_token"`
	Chunk      Bytes  `json:"chunk"`
}

type QUpdateCdnAsset struct {
	LoginToken string         `json:"login_token"`
	CdnScope   string         `json:"cdn_scope"`
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Action     CdnAssetAction `json:"action"`
}

type QUpdatePartners struct {
	LoginToken string        `json:"login_token"`
	Action     PartnerAction `json:"action"`
}

type QUpdateChangelog struct {
	LoginToken string          `json:"login_token"`
	Action     ChangelogAction `json:"action"`
}

type QUpdateBlog struct {
	LoginToken string     `json:"login_token"`
	Action     BlogAction `json:"action"`
}

type QUpdateStaffPositions struct {
	LoginToken string              `json:"login_token"`
	Action     StaffPositionAction `json:"action"`
}

type QUpdateStaffMembers struct {
	LoginToken string            `json:"login_token"`
	Action     StaffMemberAction `json:"action"`
}

type QUpdateStaffDisciplinaryType struct {
	LoginToken string                      `json:"login_token"`
	Action     StaffDisciplinaryTypeAction `json:"action"`
}

type QUpdateVoteCreditTiers struct {
	LoginToken string               `json:"login_token"`
	Action     VoteCreditTierAction `json:"action"`
}

type QUpdateShopItems struct {
	LoginToken string         `json:"login_token"`
	Action     ShopItemAction `json:"action"`
}

type QUpdateShopItemBenefits struct {
	LoginToken string                `json:"login_token"`
	Action     ShopItemBenefitAction `json:"action"`
}

type QUpdateShopCoupons struct {
	LoginToken string           `json:"login_token"`
	Action     ShopCouponAction `json:"action"`
}

type QUpdateBotWhitelist struct {
	LoginToken string             `json:"login_token"`
	Action     BotWhitelistAction `json:"action"`
}

type QPopplioStaff struct {
	LoginToken string `json:"login_token"`
	Path       string `json:"path"`
	Method     string `json:"method"`
	Body       string `json:"body"`
}

func (q *PanelQuery) UnmarshalJSON(data []byte) error {
	*q = PanelQuery{}

	return unmarshalUnion("PanelQuery", data, map[string]func() any{
		"Authorize": func() any {
			q.Authorize = &QAuthorize{}
			return q.Authorize
		},
		"Hello": func() any {
			q.Hello = &QHello{}
			return q.Hello
		},
		"BaseAnalytics": func() any {
			q.BaseAnalytics = &QLoginTokenOnly{}
			return q.BaseAnalytics
		},
		"GetUser": func() any {
			q.GetUser = &QGetUser{}
			return q.GetUser
		},
		"BotQueue": func() any {
			q.BotQueue = &QLoginTokenOnly{}
			return q.BotQueue
		},
		"ExecuteRpc": func() any {
			q.ExecuteRpc = &QExecuteRpc{}
			return q.ExecuteRpc
		},
		"GetRpcMethods": func() any {
			q.GetRpcMethods = &QGetRpcMethods{}
			return q.GetRpcMethods
		},
		"GetRpcLogEntries": func() any {
			q.GetRpcLogEntries = &QLoginTokenOnly{}
			return q.GetRpcLogEntries
		},
		"SearchEntitys": func() any {
			q.SearchEntitys = &QSearchEntitys{}
			return q.SearchEntitys
		},
		"UploadCdnFileChunk": func() any {
			q.UploadCdnFileChunk = &QUploadCdnFileChunk{}
			return q.UploadCdnFileChunk
		},
		"ListCdnScopes": func() any {
			q.ListCdnScopes = &QLoginTokenOnly{}
			return q.ListCdnScopes
		},
		"GetMainCdnScope": func() any {
			q.GetMainCdnScope = &QLoginTokenOnly{}
			return q.GetMainCdnScope
		},
		"UpdateCdnAsset": func() any {
			q.UpdateCdnAsset = &QUpdateCdnAsset{}
			return q.UpdateCdnAsset
		},
		"UpdatePartners": func() any {
			q.UpdatePartners = &QUpdatePartners{}
			return q.UpdatePartners
		},
		"UpdateChangelog": func() any {
			q.UpdateChangelog = &QUpdateChangelog{}
			return q.UpdateChangelog
		},
		"UpdateBlog": func() any {
			q.UpdateBlog = &QUpdateBlog{}
			return q.UpdateBlog
		},
		"UpdateStaffPositions": func() any {
			q.UpdateStaffPositions = &QUpdateStaffPositions{}
			return q.UpdateStaffPositions
		},
		"UpdateStaffMembers": func() any {
			q.UpdateStaffMembers = &QUpdateStaffMembers{}
			return q.UpdateStaffMembers
		},
		"UpdateStaffDisciplinaryType": func() any {
			q.UpdateStaffDisciplinaryType = &QUpdateStaffDisciplinaryType{}
			return q.UpdateStaffDisciplinaryType
		},
		"UpdateVoteCreditTiers": func() any {
			q.UpdateVoteCreditTiers = &QUpdateVoteCreditTiers{}
			return q.UpdateVoteCreditTiers
		},
		"UpdateShopItems": func() any {
			q.UpdateShopItems = &QUpdateShopItems{}
			return q.UpdateShopItems
		},
		"UpdateShopItemBenefits": func() any {
			q.UpdateShopItemBenefits = &QUpdateShopItemBenefits{}
			return q.UpdateShopItemBenefits
		},
		"UpdateShopCoupons": func() any {
			q.UpdateShopCoupons = &QUpdateShopCoupons{}
			return q.UpdateShopCoupons
		},
		"UpdateBotWhitelist": func() any {
			q.UpdateBotWhitelist = &QUpdateBotWhitelist{}
			return q.UpdateBotWhitelist
		},
		"PopplioStaff": func() any {
			q.PopplioStaff = &QPopplioStaff{}
			return q.PopplioStaff
		},
	})
}

func (q PanelQuery) MarshalJSON() ([]byte, error) {
	switch {
	case q.Authorize != nil:
		return encodeVariant("Authorize", q.Authorize)
	case q.Hello != nil:
		return encodeVariant("Hello", q.Hello)
	case q.BaseAnalytics != nil:
		return encodeVariant("BaseAnalytics", q.BaseAnalytics)
	case q.GetUser != nil:
		return encodeVariant("GetUser", q.GetUser)
	case q.BotQueue != nil:
		return encodeVariant("BotQueue", q.BotQueue)
	case q.ExecuteRpc != nil:
		return encodeVariant("ExecuteRpc", q.ExecuteRpc)
	case q.GetRpcMethods != nil:
		return encodeVariant("GetRpcMethods", q.GetRpcMethods)
	case q.GetRpcLogEntries != nil:
		return encodeVariant("GetRpcLogEntries", q.GetRpcLogEntries)
	case q.SearchEntitys != nil:
		return encodeVariant("SearchEntitys", q.SearchEntitys)
	case q.UploadCdnFileChunk != nil:
		return encodeVariant("UploadCdnFileChunk", q.UploadCdnFileChunk)
	case q.ListCdnScopes != nil:
		return encodeVariant("ListCdnScopes", q.ListCdnScopes)
	case q.GetMainCdnScope != nil:
		return encodeVariant("GetMainCdnScope", q.GetMainCdnScope)
	case q.UpdateCdnAsset != nil:
		return encodeVariant("UpdateCdnAsset", q.UpdateCdnAsset)
	case q.UpdatePartners != nil:
		return encodeVariant("UpdatePartners", q.UpdatePartners)
	case q.UpdateChangelog != nil:
		return encodeVariant("UpdateChangelog", q.UpdateChangelog)
	case q.UpdateBlog != nil:
		return encodeVariant("UpdateBlog", q.UpdateBlog)
	case q.UpdateStaffPositions != nil:
		return encodeVariant("UpdateStaffPositions", q.UpdateStaffPositions)
	case q.UpdateStaffMembers != nil:
		return encodeVariant("UpdateStaffMembers", q.UpdateStaffMembers)
	case q.UpdateStaffDisciplinaryType != nil:
		return encodeVariant("UpdateStaffDisciplinaryType", q.UpdateStaffDisciplinaryType)
	case q.UpdateVoteCreditTiers != nil:
		return encodeVariant("UpdateVoteCreditTiers", q.UpdateVoteCreditTiers)
	case q.UpdateShopItems != nil:
		return encodeVariant("UpdateShopItems", q.UpdateShopItems)
	case q.UpdateShopItemBenefits != nil:
		return encodeVariant("UpdateShopItemBenefits", q.UpdateShopItemBenefits)
	case q.UpdateShopCoupons != nil:
		return encodeVariant("UpdateShopCoupons", q.UpdateShopCoupons)
	case q.UpdateBotWhitelist != nil:
		return encodeVariant("UpdateBotWhitelist", q.UpdateBotWhitelist)
	case q.PopplioStaff != nil:
		return encodeVariant("PopplioStaff", q.PopplioStaff)
	default:
		return nil, fmt.Errorf("PanelQuery: no variant set")
	}
}
