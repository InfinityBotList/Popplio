# Ratelimits
Our API implements a process for limiting and preventing spam requests. 
API users that regularly hit and ignore the limit will be blocked from our platform. 
These rate limits are in place to help prevent the abuse and overload of our services.

--- 

**As of API v6, all ratelimits including global ratelimits are dynamic and can be added/removed at *any* time as we require.**

Finally, some other pointers:

- Not all endpoints may return these headers however these may still have ratelimits.
- A number for the purpose of the below table is defined as a number stringified

## Rate Limit Header Structure

| Field                  | Type        | Description                                                                                         |
| ---------------------- | ----------- | --------------------------------------------------------------------------------------------------- |
| X-Ratelimit-Bucket      | String ("abc")   | The bucket you are going to performing the request on. Different requests have different buckets                                                        |
| X-Ratelimit-Bucket-Reqs-Allowed-Count  | Integer (50)    | Number of requests allowed per time interval                                                   |
| X-Ratelimit-Bucket-Reqs-Allowed-Second      | Integer   | Time interval between ratelimit resets                                              |
| X-Ratelimit-Req-Made | Integer | The number of requests you have made in the time interval |
| retry-afer             | Integer    | Amount of time until the Rate Limit Expires. Learn more about this header [here](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After)  |

---

## Rate Limit Response Structure

| Field     | Type        | Description                                                                                         |
| --------- | ----------- | --------------------------------------------------------------------------------------------------- |
| message   | `String`    | Error Message for the Rate Limit Response. This is a constant as per below and is static                                                     |

## Example Rate Limit Response
```json
{
    "message": "You're being rate limited!"
}
```

Clients are expected to use the ``retry-after`` header. Ratelimit responses are now static to improve performance.

Furhter more, some endpoints have what is called a bypass bucket. Handle these the same way you would treat normal ratelimits.