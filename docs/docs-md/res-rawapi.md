# Javascript/Python (Raw API)
If you prefer to interact with our raw API instead of using a Third Party Module
you can follow the usage guides and examples below which should be more then enough to 
get you started xD

> NOTE: For posting stats you can send up to `3` requests every `5 Minutes`

---

## Javascript Usage

**Post Stats**

```js
const fetch = require("node-fetch")
fetch(`{apiUrl}/bots/stats`, {
    method: "POST",
    headers: {
        "authorization": "api-key-here",
        "Content-Type": "application/json"
    },
    body: JSON.stringify({
        servers: 100,
        shards: 69
    })
}).then(res => res.json())
.then(json => console.log(json))
```

**Get Bot**

```js
const fetch = require("node-fetch")
fetch(`{apiUrl}/bots/:botID`, {
    method: "GET",
    headers: {
        "Content-Type": "application/json"
    }
}).then(async res => console.log(await res.json()));
```

---

## Python Usage

```python
import aiohttp

# In a async function
async def post_stats():
    async with aiohttp.ClientSession() as sess:
        headers= {
            "authorization": "Your api token",       
        }
        payload= {
            "servers": len(bot.guilds) or 0, # Change this if you use custom clustering
            "shards": bot.shard_count or 0   # Change this if you use custom clustering
        }
        async with sess.post("{apiUrl}/bots/stats", headers=headers, json=payload) as res:
            # Do something with the response. EX
            # if res.status == 200:
            # ...
```
