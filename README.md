# Popplio

Popplio is the new rewrite of the Infinity Bot List API in golang

It is open source for transparency but we do not support self-hosting of this whatsoever

## Endpoints

``/`` => Index page

``/bots/{id}`` => Get bot

- Set ``resolve`` to ``true`` (or ``1``) to also resolve bot name/vanity in this endpoint
- Returns a ``types.Bot`` object

``/users/{uid}/bots/{bid}/votes`` => Get User Votes

- Requires authentication
- Returns a ``[]uint64`` where each ``uint64`` represents a timestamp of the vote

``/bots/stats`` | ``/bots/{id}/stats`` => Post Stats

- Requires authentication
- Accepts methods ``PUT/POST/PATCH`` (they all do the same thing)
- Send either ``count`` in query parameters or send a JSON body. Most combinations of keys used for server/shard count will work including the d (``servers``, ``shard_count`` way
