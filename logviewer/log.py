#!/usr/bin/python3
from io import TextIOWrapper
import subprocess
from typing import Sequence
import fastapi
from fastapi.responses import ORJSONResponse
import orjson
import uvicorn
import secrets
from dateutil.parser import parse

psk = secrets.token_hex(128)

app = fastapi.FastAPI()

def_headers = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "PSK",
}

def line_count(filename: str):
    return int(subprocess.check_output(['wc', '-l', filename]).split()[0])

def char_check(s: str) -> bool:
    """Returns true if string is alphanumeric or a dot"""
    return all(c.isalnum() or c == "-" or c == '.' for c in s)

class FilteredFile():
    def __init__(
        self, 
        fobj: TextIOWrapper, 
        allowed_levels: Sequence[str] = []
    ):
        self.fobj = fobj
        self.allowed_levels = allowed_levels
    
    def __iter__(self):
        for _line in self.fobj:
            
            line = orjson.loads(_line)

            if line.get("level"):
                line["level"] = line["level"].lower()

            if line.get("ts"):
                if isinstance(line.get("ts"), str):
                    line["ts"] = parse(line["ts"]).timestamp()

            if not self.allowed_levels:
                yield line
            else:
                if line.get("level") in self.allowed_levels:
                    yield line
    
    def __len__(self):
        return sum(1 for _ in self)

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
async def length(fn: str, request: fastapi.Request, allowed_levels: list[str] | None = None):
    if request.headers.get("PSK") != psk:
        return ORJSONResponse({"error": "invalid psk"}, status_code=403, headers=def_headers)

    # Proxy protection
    if request.headers.get("X-Forwarded-For"):
        print("X-Forwarded-For: ", request.headers.get("X-Forwarded-For"))
        return fastapi.Response(status_code=400)
    
    # prevent logviewer from being publicly accessible
    if request.url.scheme == "https" or request.url.port != 1039:
        return fastapi.Response(status_code=400)

    # Ensure fn is only alphanumeric characters or period
    if not char_check(fn):
        return fastapi.Response(status_code=400)

    if not allowed_levels:
        # Fast happy path
        return ORJSONResponse({"length": line_count(f"/var/log/{fn}")}, headers=def_headers)

    # Slow path since filters are applied
    with open(f"/var/log/{fn}") as f:
        return ORJSONResponse({"length": len(FilteredFile(f, allowed_levels))}, headers=def_headers)

@app.get("/{fn}")
async def read_item(request: fastapi.Request, fn: str, limit: int, offset: int, allowed_levels: list[str] | None = None):
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

    # Ensure fn is only alphanumeric characters or period
    if not char_check(fn):
        return fastapi.Response(status_code=400)

    with open(f"/var/log/{fn}") as json_file:
        # Lazy load the json file to a list
        read = 0

        json_list = []

        for v in FilteredFile(json_file, allowed_levels or []):
            if read >= offset:
                json_list.append(v)
                if len(json_list) >= limit:
                    break
            read += 1
    
    return ORJSONResponse(json_list, headers=def_headers)

print(f"PSK for logviewer: {psk}")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=1039)  # type: ignore  # type: ignore)
