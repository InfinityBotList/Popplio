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

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("PanelQuery: %w", err)
	}

	// Every operation is a struct variant, so none accept the bare-string form.
	var into any

	switch name {
	case "Authorize":
		q.Authorize = &QAuthorize{}
		into = q.Authorize
	case "Hello":
		q.Hello = &QHello{}
		into = q.Hello
	case "BaseAnalytics":
		q.BaseAnalytics = &QLoginTokenOnly{}
		into = q.BaseAnalytics
	case "GetUser":
		q.GetUser = &QGetUser{}
		into = q.GetUser
	case "BotQueue":
		q.BotQueue = &QLoginTokenOnly{}
		into = q.BotQueue
	case "ExecuteRpc":
		q.ExecuteRpc = &QExecuteRpc{}
		into = q.ExecuteRpc
	case "GetRpcMethods":
		q.GetRpcMethods = &QGetRpcMethods{}
		into = q.GetRpcMethods
	case "GetRpcLogEntries":
		q.GetRpcLogEntries = &QLoginTokenOnly{}
		into = q.GetRpcLogEntries
	case "SearchEntitys":
		q.SearchEntitys = &QSearchEntitys{}
		into = q.SearchEntitys
	case "UpdatePartners":
		q.UpdatePartners = &QUpdatePartners{}
		into = q.UpdatePartners
	case "UpdateChangelog":
		q.UpdateChangelog = &QUpdateChangelog{}
		into = q.UpdateChangelog
	case "UpdateBlog":
		q.UpdateBlog = &QUpdateBlog{}
		into = q.UpdateBlog
	case "UpdateStaffPositions":
		q.UpdateStaffPositions = &QUpdateStaffPositions{}
		into = q.UpdateStaffPositions
	case "UpdateStaffMembers":
		q.UpdateStaffMembers = &QUpdateStaffMembers{}
		into = q.UpdateStaffMembers
	case "UpdateStaffDisciplinaryType":
		q.UpdateStaffDisciplinaryType = &QUpdateStaffDisciplinaryType{}
		into = q.UpdateStaffDisciplinaryType
	case "UpdateVoteCreditTiers":
		q.UpdateVoteCreditTiers = &QUpdateVoteCreditTiers{}
		into = q.UpdateVoteCreditTiers
	case "UpdateShopItems":
		q.UpdateShopItems = &QUpdateShopItems{}
		into = q.UpdateShopItems
	case "UpdateShopItemBenefits":
		q.UpdateShopItemBenefits = &QUpdateShopItemBenefits{}
		into = q.UpdateShopItemBenefits
	case "UpdateShopCoupons":
		q.UpdateShopCoupons = &QUpdateShopCoupons{}
		into = q.UpdateShopCoupons
	case "UpdateBotWhitelist":
		q.UpdateBotWhitelist = &QUpdateBotWhitelist{}
		into = q.UpdateBotWhitelist
	case "PopplioStaff":
		q.PopplioStaff = &QPopplioStaff{}
		into = q.PopplioStaff
	default:
		return errUnknownVariant("PanelQuery", name)
	}

	return decodeVariant("PanelQuery", name, payload, into)
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
