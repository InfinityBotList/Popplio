# This should be run on a residential system due to imgur rate limiting
# E.g. python3 -m uvicorn imgproxy:app --host 0.0.0.0

import ipaddress
import socket
from urllib.parse import urlparse

import aiohttp
import fastapi
from yarl import URL

app = fastapi.FastAPI()

MAX_REDIRECTS = 5

ACCEPT_HEADER = "image/png,image/webp,image/apng,image/*,*/*;q=0.8"


def _is_unsafe_address(host: str) -> bool:
    """Resolves host and reports whether any of its addresses point at a
    private, loopback, link-local (this includes cloud metadata endpoints
    like 169.254.169.254), or otherwise reserved range.

    Every candidate address is checked, not just the first, since a
    hostname can resolve to a mix of public and internal addresses.
    """
    try:
        infos = socket.getaddrinfo(host, None)
    except socket.gaierror:
        # Unresolvable is safe to reject outright rather than let the
        # underlying request fail with a less clear error.
        return True

    for info in infos:
        addr = info[4][0]
        try:
            ip = ipaddress.ip_address(addr)
        except ValueError:
            return True

        if ip.is_private or ip.is_loopback or ip.is_link_local or ip.is_reserved or ip.is_multicast or ip.is_unspecified:
            return True

    return False


def validate_target(url: str) -> None:
    """Rejects anything that isn't a plain http(s) URL pointing at a public
    address. Raises fastapi.HTTPException(400) on rejection.

    This is a resolve-then-check (not connect-pinned) validation, so a
    DNS answer that changes between this check and the actual request
    (DNS rebinding) isn't fully closed off. That residual risk is judged
    acceptable here: this tool isn't meant to be reachable from the open
    internet in the first place (see the module comment), and this check
    still closes the straightforward "pass a private/link-local/metadata
    URL directly" attack CodeQL flagged.
    """
    parsed = urlparse(url)

    if parsed.scheme not in ("http", "https"):
        raise fastapi.HTTPException(status_code=400, detail="URL must be http or https")

    if not parsed.hostname:
        raise fastapi.HTTPException(status_code=400, detail="URL must have a host")

    if _is_unsafe_address(parsed.hostname):
        raise fastapi.HTTPException(status_code=400, detail="URL host is not allowed")


async def fetch_validated(method: str, url: str) -> aiohttp.ClientResponse:
    """Performs the request with redirects disabled, manually following and
    re-validating each hop so a redirect can't retarget the request at a
    private/internal address after the initial URL passed validation.
    """
    validate_target(url)

    session = aiohttp.ClientSession()
    try:
        current = url
        for _ in range(MAX_REDIRECTS + 1):
            resp = await session.request(
                method,
                current,
                allow_redirects=False,
                headers={"Accept": ACCEPT_HEADER},
            )

            if resp.status in (301, 302, 303, 307, 308) and "Location" in resp.headers:
                location = resp.headers["Location"]
                resp.close()

                next_url = str(URL(current).join(URL(location)))
                validate_target(next_url)
                current = next_url
                continue

            return resp

        raise fastapi.HTTPException(status_code=400, detail="Too many redirects")
    finally:
        await session.close()


@app.head("/")
async def head(url: str):
    resp = await fetch_validated("HEAD", url)
    try:
        return fastapi.responses.PlainTextResponse(
            content="",
            status_code=resp.status,
            headers={name: value for name, value in resp.headers.items()},
        )
    finally:
        resp.close()


@app.get("/")
async def get(url: str):
    resp = await fetch_validated("GET", url)
    try:
        resp_bytes = await resp.read()
        return fastapi.responses.Response(
            content=resp_bytes, media_type=resp.headers.get("Content-Type", "image/png")
        )
    finally:
        resp.close()
