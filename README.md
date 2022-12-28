# Popplio

Popplio is the new rewrite of the Infinity Bot List API in golang

**Open source under the AGPL3. We reserve all rights to the code**

## API Docs

https://spider.infinitybots.gg/docs

## Developer Docs

There is a tool coming very soon (``ibl newroute``) to assist in creating new endpoints on Popplio

- Whenever you need to fetch a user from discord, always use ``utils.GetDiscordUser`` as that also handles caching (both gateway and redis and internal caches)
