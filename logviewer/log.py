#!/usr/bin/python3
import subprocess
import fastapi
from fastapi.responses import ORJSONResponse
import orjson
import uvicorn
import secrets

psk = secrets.token_hex(128)

app = fastapi.FastAPI()

def_headers = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "PSK",
}

def line_count(filename: str):
    return int(subprocess.check_output(['wc', '-l', filename]).split()[0])

@app.options("/{fn}")
async def options(fn: str):
    return ORJSONResponse({}, headers=def_headers)

@app.options("/conntest/a")
async def options_ctest():
    return ORJSONResponse({}, headers=def_headers)

@app.get("/conntest/a")
async def conntest(request: fastapi.Request):
    if request.headers.get("PSK") != psk:
        return ORJSONResponse({"error": "invalid psk"}, status_code=403, headers=def_headers)

    return ORJSONResponse({}, headers=def_headers)

@app.options("/{fn}/length")
async def options_length(fn: str):
    return ORJSONResponse({}, headers=def_headers)

@app.get("/{fn}/length")
async def length(fn: str, request: fastapi.Request):
    if request.headers.get("PSK") != psk:
        return ORJSONResponse({"error": "invalid psk"}, status_code=403, headers=def_headers)

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

    return ORJSONResponse({"length": line_count(f"/var/log/{fn}")}, headers=def_headers)

@app.get("/{fn}")
async def read_item(request: fastapi.Request, fn: str, limit: int, offset: int):
    if request.headers.get("PSK") != psk:
        return ORJSONResponse({"error": "invalid psk"}, status_code=403, headers=def_headers)
    
    if limit > 300:
        return ORJSONResponse({"error": "limit too large"}, status_code=400, headers=def_headers)
    
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

        json_list = []

        for v in json_file:
            if read >= offset:
                json_list.append(orjson.loads(v))
                if len(json_list) >= limit:
                    break
            read += 1
            print(read)
    
    return ORJSONResponse(json_list, headers=def_headers)

print(f"PSK for logviewer: {psk}")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=1039)  # type: ignore  # type: ignore)