# This should be run on a residential system due to imgur rate limiting
# E.g. python3 -m uvicorn imgproxy:app --host 0.0.0.0

import aiohttp
import fastapi

app = fastapi.FastAPI()

@app.head("/")
async def head(url: str):
    async with aiohttp.ClientSession() as session:
        async with session.head(
            url,
            allow_redirects=True,  
            headers={
                #"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
                "Accept": "image/png,image/webp,image/apng,image/*,*/*;q=0.8",
        }) as resp:
            return fastapi.responses.PlainTextResponse(
                content="",
                status_code=resp.status,
                headers={
                    name: value
                    for name, value in resp.headers.items()
                },
            )

@app.get("/")
async def get(url: str):
    async with aiohttp.ClientSession() as session:
        async with session.get(
            url,
            allow_redirects=True, 
            headers={
                #"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/
                "Accept": "image/png,image/webp,image/apng,image/*,*/*;q=0.8",
        }) as resp:
            # Convert to file-like object
            resp_bytes = await resp.read()

            # Return response as a stream
            return fastapi.responses.Response(content=resp_bytes, media_type=resp.headers.get("Content-Type", "image/png"))
