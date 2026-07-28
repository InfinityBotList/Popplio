import requests, asyncpg, asyncio

async def main():
    conn = await asyncpg.connect('postgresql:///infinity')

    uids = await conn.fetch('SELECT user_id FROM users')

    for uid in uids:
        print(f"Updating CDN for user {uid['user_id']}...")

        # Call DELETE first to clear cache
        resp = requests.delete(f"https://spider-staging.infinitybots.gg/platform/user/{uid['user_id']}?platform=discord")

        if not resp.ok:
            print(f"Failed to clear cache for user {uid['user_id']}")
        
        # Call GET to update cache
        resp = requests.get(f"https://spider-staging.infinitybots.gg/platform/user/{uid['user_id']}?platform=discord")

        if not resp.ok:
            print(f"Failed to update cache for user {uid['user_id']}")

        await asyncio.sleep(0.5)

    await conn.close()

asyncio.run(main())