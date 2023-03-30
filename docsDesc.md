# Introduction

Hey there 👋, welcome to our Official Documentation!

---

## Getting Help

**Please see https://docs.botlist.site for more info on the basics of our API. This site purely exists to be an API reference not a guide**

If you need some help or think you have spotted a problem with our API you can talk to us in our 
[#api-support](https://discord.com/channels/758641373074423808/826363644295643136) channel in our [discord server](https://infinitybotlist.com/discord).	

## Parsing webhooks

To parse webhooks, here is the algorithm you should/must follow:

Examples are provided currently in JS but will be added for other languages in the future.

- Check the protocol version:
	- The current protocol version is `splashtail`
	- Check the `X-Webhook-Protocol` header and ensure that it is equal to the current protocol version

<div class="javascript">

```js
  if (req.headers["x-webhook-protocol"] != supportedProtocol) {
    reply.status(403).send({
      message: "Invalid protocol version!",
      error: true,
    });
    return;
  }
```

</div>

- A nonce is used to randomize the signature for retries. Ensure a nonce exists by checking the header's existence:

<div class="javascript">

```js
  if (!req.headers["x-webhook-nonce"]) {
    reply.status(403).send({
      message: "No nonce provided?",
      error: true,
    });
    return;
  }
```

</div>

- Next calculate the expected signature
	- To do so, you must first get the body of the request
	- Then use HMAC-SHA512 with the webhook secret as key and the body as the request body to get the ``signedBody``. Note that the format/digest should be ``hex``
	- Then use HMAC-SHA512 with the nonce as the key and the signed body as the message to get the expected signature. Note that the format/digest should be ``hex``

<div class="javascript">

```js
  let body: string = req.body;

  if (!body) {
    reply.status(400).send({
      message: "No request body provided?",
      error: true,
    });
    return;
  }

  // Create hmac 512 hash
  let signedBody = crypto
    .createHmac("sha512", webhookSecret)
    .update(body)
    .digest("hex");

  // Create the actual signature using x-webhook-nonce by performing a second hmac
  let nonce = req.headers["x-webhook-nonce"].toString();
  let expectedTok = crypto
    .createHmac("sha512", nonce)
    .update(signedBody)
    .digest("hex");
```

</div>

- Compare this value with the ``X-Webhook-Signature`` header
	- If they are equal, the request is valid and you can continue processing it
	- If they are not equal, the request is invalid and you should return a 403 status code

<div class="javascript">

```js
  if (req.headers["x-webhook-signature"] != expectedTok) {
    console.log(
      `Expected: ${expectedTok} Got: ${req.headers["x-webhook-signature"]}`
    );
    reply.status(403).send({
      message: "Invalid signature",
      error: true,
    });
    return;
  }
```

</div>

- Next decrypt the request body. This is an additional security to prevent sensitive information from being leaked
	- First hash the concatenation of the webhook secret and the nonce using SHA256
	- Then read the body as a hex string and decrypt it using AES-256-GCM with the hashed secret as the key

<div class="javascript">

```js
	// sha256 on key
	let hashedKey = crypto
	.createHash("sha256")
	.update(webhookSecret + nonce)
	.digest();

	let enc = Buffer.from(body, "hex");
	const tag = enc.subarray(enc.length - tagLength, enc.length);
	const iv = enc.subarray(0, ivLength);
	const toDecrypt = enc.subarray(ivLength, enc.length - tag.length);
	const decipher = crypto.createDecipheriv("aes-256-gcm", hashedKey, iv);
	decipher.setAuthTag(tag);
	const res = Buffer.concat([decipher.update(toDecrypt), decipher.final()]);

	// Parse the decrypted body
	let data = JSON.parse(res.toString("utf-8"));

	if (data.created_at == undefined) {
	reply.status(400).send({
	message: "Invalid body",
	error: true,
	});
	return;
```

</div>
