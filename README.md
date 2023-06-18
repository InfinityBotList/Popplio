# Popplio

Popplio is the new rewrite of the Infinity Bot List API in golang...

**Open source under the AGPL3. We reserve all rights to the code**

## API Docs

https://spider.infinitybots.gg/docs

**Quick Note**

- Whenever you need to fetch a user from discord, always use ``dovewing.GetUser`` as that also handles caching (both gateway and redis and internal caches)

## Creating a config

You can use ``./popplio --cmd genconfig`` to create a configuration file for Popplio
