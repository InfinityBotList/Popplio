package sender

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"

	"popplio/state"

	"github.com/infinitybotlist/eureka/crypto"
	"go.uber.org/zap"
)

// userAgent identifies Popplio to webhook endpoints.
const userAgent = "Popplio/v8.0.0 (https://infinitybots.gg)"

// dnsTimeout bounds the target hostname lookup. A webhook whose DNS is slow
// should not hold a delivery slot open.
const dnsTimeout = 5 * time.Second

// webhookClient performs webhook deliveries. The timeout bounds how long a slow
// or hostile endpoint can occupy a sender.
var webhookClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Secret is a webhook's shared secret, used to authenticate payloads.
type Secret struct {
	Raw string
}

// Sign returns the hex-encoded HMAC-SHA512 of data under this secret.
func (s Secret) Sign(data []byte) string {
	h := hmac.New(sha512.New, []byte(s.Raw))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// resolveTarget resolves the webhook URL and rejects addresses Popplio must not
// connect to.
//
// Webhook URLs are user-supplied, so delivering to one is a request to make an
// outbound connection to an arbitrary host — the classic SSRF shape. Resolving
// up front and caching the result on the send state means the addresses are
// vetted once and reused, rather than re-resolved per attempt where a hostile
// DNS server could answer differently the second time.
//
// Resolution is skipped when the state already carries addresses, which is how
// the bad-intent probe inherits the vetted set from the delivery that spawned it.
func (st *webhookSendState) resolveTarget(webhookURL string) error {
	if len(st.ResolvedIps) == 0 {
		parsed, err := url.ParseRequestURI(webhookURL)

		if err != nil {
			st.cancelSend("INVALID_REQUEST_URL")
			return err
		}

		timeoutCtx, cancel := context.WithTimeout(state.Context, dnsTimeout)
		defer cancel()

		ips, err := net.DefaultResolver.LookupHost(timeoutCtx, parsed.Hostname())

		if err != nil {
			st.cancelSend("CNAME_LOOKUP_FAILURE")
			return err
		}

		st.ResolvedIps = ips
	}

	state.Logger.Info("Resolved webhook IP", st.logFields(zap.Strings("resolvedIp", st.ResolvedIps))...)

	if slices.Contains(st.ResolvedIps, "127.0.0.1") {
		st.cancelSend("LOCALHOST_URL")
		return errors.New("localhost url")
	}

	return nil
}

// buildRequest constructs the outgoing POST for this delivery.
//
// Which secret is used depends on intent: a bad-intent probe signs with a
// throwaway secret precisely so a correctly-implemented endpoint rejects it.
//
// Two wire protocols are supported. simple-auth sends the payload as-is with the
// secret in the Authorization header, for endpoints that cannot implement more.
// splashtail encrypts the payload and sends a nonce-chained HMAC, so neither the
// body nor a captured signature can be replayed.
func (st *webhookSendState) buildRequest(webhook *webhookData, data []byte) (*http.Request, error) {
	secret := webhook.Secret

	if st.BadIntent {
		secret = crypto.RandString(128)
	}

	var req *http.Request
	var err error

	if webhook.SimpleAuth {
		req, err = http.NewRequestWithContext(state.Context, "POST", webhook.Url, bytes.NewReader(data))

		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", secret)
		req.Header.Set("X-Webhook-Protocol", "simple-auth")
	} else {
		postData, nonce, token, err := sealPayload(secret, data)

		if err != nil {
			return nil, err
		}

		req, err = http.NewRequestWithContext(state.Context, "POST", webhook.Url, bytes.NewReader(postData))

		if err != nil {
			return nil, err
		}

		req.Header.Set("X-Webhook-Signature", token)
		req.Header.Set("X-Webhook-Protocol", "splashtail")
		req.Header.Set("X-Webhook-Nonce", nonce)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", userAgent)

	return req, nil
}

// sealPayload encrypts data and signs it under the splashtail protocol.
//
// The per-request nonce does double duty: it salts the AES key so the same
// payload never encrypts identically twice, and it keys the outer HMAC so a
// signature captured from one request cannot be replayed against another.
//
// Returns the hex-encoded ciphertext to send as the body, the nonce to publish
// in the X-Webhook-Nonce header, and the signature for X-Webhook-Signature.
func sealPayload(secret string, data []byte) (postData []byte, nonce, token string, err error) {
	nonce = crypto.RandString(16)

	keyHash := sha256.New()
	keyHash.Write([]byte(secret + nonce))

	block, err := aes.NewCipher(keyHash.Sum(nil))

	if err != nil {
		return nil, "", "", err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, "", "", err
	}

	aesNonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, aesNonce); err != nil {
		return nil, "", "", err
	}

	postData = []byte(hex.EncodeToString(gcm.Seal(aesNonce, aesNonce, data, nil)))

	// Signed with the secret, then re-signed with the nonce, so verifying
	// requires both.
	inner := Secret{Raw: secret}.Sign(postData)
	token = Secret{Raw: nonce}.Sign([]byte(inner))

	return postData, nonce, token, nil
}
