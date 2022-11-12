#!/usr/bin/python3
import fastapi
from fastapi.responses import ORJSONResponse
import orjson
import uvicorn

app = fastapi.FastAPI()

@app.options("/{fn}")
async def options(fn: str):
    return ORJSONResponse({}, headers={
        "Access-Control-Allow-Origin": "*",
    })

@app.get("/{fn}")
async def read_item(request: fastapi.Request, fn: str, limit: int, offset: int):
    if limit > 300:
        return ORJSONResponse({"error": "limit too large"}, status_code=400)
    
    # Proxy protection
    if request.headers.get("X-Forwarded-For"):
        print("X-Forwarded-For: ", request.headers.get("X-Forwarded-For"))
        return fastapi.Response(status_code=400)
    
    # prevent logviewer from being publicly accessible
    if request.url.scheme == "https" or request.url.port != 1039:
        return fastapi.Response(status_code=400)

    # Ensure fn is only ascii characters or period
    if not (fn.isascii() and "." in fn):
        return fastapi.Response(status_code=400)

    with open(f"/var/log/{fn}") as json_file:
        # Lazy load the json file to a list
        read = 0
        json_list = [v for v in json_file if read < limit and (read := read + 1) >= offset]

    vals = []
    for v in json_list[int(offset):int(offset)+int(limit)]:
        vals.append(orjson.loads(v))
    
    return ORJSONResponse(vals, headers={
        "Access-Control-Allow-Origin": "*",
    })

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=1039)  # type: ignore  # type: ignore)