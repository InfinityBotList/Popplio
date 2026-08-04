package types

import "fmt"

// AuthorizeAction is the union of login-flow actions. Every variant is a struct
// variant, so none of them accept the bare-string form.
type AuthorizeAction struct {
	Begin           *AuthBegin
	CreateSession   *AuthCreateSession
	CheckMfaState   *AuthCheckMfaState
	ResetMfaTotp    *AuthResetMfaTotp
	ActivateSession *AuthActivateSession
	Logout          *AuthLogout
}

type AuthBegin struct {
	Scope       string `json:"scope"`
	RedirectURL string `json:"redirect_url"`
}

type AuthCreateSession struct {
	Code        string `json:"code"`
	RedirectURL string `json:"redirect_url"`
}

type AuthCheckMfaState struct {
	LoginToken string `json:"login_token"`
}

type AuthResetMfaTotp struct {
	LoginToken string `json:"login_token"`
	OTP        string `json:"otp"`
}

type AuthActivateSession struct {
	LoginToken string `json:"login_token"`
	OTP        string `json:"otp"`
}

type AuthLogout struct {
	LoginToken string `json:"login_token"`
}

func (a *AuthorizeAction) UnmarshalJSON(data []byte) error {
	*a = AuthorizeAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("AuthorizeAction: %w", err)
	}

	// Every variant is a struct variant, so none accept the bare-string form.
	var into any

	switch name {
	case "Begin":
		a.Begin = &AuthBegin{}
		into = a.Begin
	case "CreateSession":
		a.CreateSession = &AuthCreateSession{}
		into = a.CreateSession
	case "CheckMfaState":
		a.CheckMfaState = &AuthCheckMfaState{}
		into = a.CheckMfaState
	case "ResetMfaTotp":
		a.ResetMfaTotp = &AuthResetMfaTotp{}
		into = a.ResetMfaTotp
	case "ActivateSession":
		a.ActivateSession = &AuthActivateSession{}
		into = a.ActivateSession
	case "Logout":
		a.Logout = &AuthLogout{}
		into = a.Logout
	default:
		return errUnknownVariant("AuthorizeAction", name)
	}

	return decodeVariant("AuthorizeAction", name, payload, into)
}

func (a AuthorizeAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.Begin != nil:
		return encodeVariant("Begin", a.Begin)
	case a.CreateSession != nil:
		return encodeVariant("CreateSession", a.CreateSession)
	case a.CheckMfaState != nil:
		return encodeVariant("CheckMfaState", a.CheckMfaState)
	case a.ResetMfaTotp != nil:
		return encodeVariant("ResetMfaTotp", a.ResetMfaTotp)
	case a.ActivateSession != nil:
		return encodeVariant("ActivateSession", a.ActivateSession)
	case a.Logout != nil:
		return encodeVariant("Logout", a.Logout)
	default:
		return nil, fmt.Errorf("AuthorizeAction: no variant set")
	}
}

// MfaLoginSecret is returned when the caller still needs to enrol in TOTP.
type MfaLoginSecret struct {
	Secret string `json:"secret"`
	OtpURL string `json:"otp_url"`
	QRCode string `json:"qr_code"`
}

// MfaLogin carries a null `info` when the caller is already enrolled.
type MfaLogin struct {
	Info *MfaLoginSecret `json:"info"`
}

// AuthData is the session descriptor returned by the auth checks and embedded in
// the Hello response.
type AuthData struct {
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	State     string `json:"state"`
}
