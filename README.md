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
- Returns: 

	- ts: ``[]uint64`` where each ``uint64`` represents a timestamp of the vote *or* a 404 if no votes were made

	- vote_time: ``uint16`` which is the time a user has to wait between two votes (currently 12 hours). This value is in hours

	- has_voted: ``bool`` stating whether or not this user has voted for your bot in the past 12 hours

``/bots/stats`` | ``/bots/{id}/stats`` => Post Stats

- Requires authentication
- Accepts methods ``PUT/POST/PATCH`` (they all do the same thing)
- Send either ``count`` in query parameters or send a JSON body. Most combinations of keys used for server/shard count will work including the d (``servers``, ``shard_count`` way


**Endpoints under ``siteinternal.go`` are internal and are not meant to be used. They *can* change at any time randomly**