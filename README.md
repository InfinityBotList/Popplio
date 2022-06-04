# Popplio

Popplio is the new rewrite of the Infinity Bot List API in golang

It is open source for transparency but we do not support self-hosting of this whatsoever

## Endpoints

``/`` => Index page

``/bots/{id}`` => Get bot
    - Set ``resolve`` to ``true`` (or ``1``) to also resolve bot name/vanity in this endpoint