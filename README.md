# Popplio

Popplio is the new rewrite of the Infinity Bot List API in golang

It is open source for transparency but we do not support self-hosting of this whatsoever

**You may use the metro.go file freely as a example of a metro integration under the MIT**

## User Tokens

Some APIs on Popplio require a user token. To get one, go to profile settings > View Token

## Endpoints

``/`` => Index page

``/bots/{id}`` => Get Bot

- Set ``resolve`` to ``true`` (or ``1``) to also resolve bot name/vanity in this endpoint
- Returns a ``types.Bot`` object
- Responses are cached for 3 minutes, the ``x-popplio-cached`` header will be set to ``true`` in this case

``/users/{id}`` => Get User

- Set ``resolve`` to ``true`` (or ``1``) to also resolve user nickname in this endpoint
- Returns a ``types.User`` object
- Responses are cached for 3 minutes, the ``x-popplio-cached`` header will be set to ``true`` in this case

``/bots/{id}/reviews`` => Get bot reviews

- Returns a ``types.Review`` object

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
