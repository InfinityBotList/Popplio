# Introduction

Hey there 👋, welcome to our Official Documentation!

---

#### Navigation

If you want to just see the list of endpoints, click [here](#operations-top).

#### I'm busy son!

We offer several libraries for interacting with the API as seen below:

## Getting Help

If you need some help or think you have spotted a problem with our API you can talk to us in our 
[`#api-support`](https://discord.com/channels/758641373074423808/826363644295643136) channel in our [discord server](https://infinitybotlist.com/discord).

---

## API Intro

Infinity uses a REST(ish) API for most of its functionality. This API is used by our website and our bots to interact with the database.

## Authorization

To access our API you need to authorize yourself or in this case your bot, this can be done by using your Infinity API Token which can be found in the `Owner Section` of your bots page.

![Owner Section Screenshot](https://media.discordapp.net/attachments/832011830238248961/871632845821591562/image0.png)

Authentication is performed with the `authorization` HTTP header:

```
Header: your-token-here with prefix
```

## Please see https://docs.botlist.site for more info on the basics of our API. This site purely exists to be an API reference not a guide

