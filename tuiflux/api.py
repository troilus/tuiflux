import httpx
from typing import List, Dict, Optional
from .models import Feed, Entry

class MinifluxAPI:
    def __init__(self, server_url: str, api_key: str, verify_ssl: bool = True):
        self.server_url = server_url
        self.api_key = api_key
        self.headers = {"X-Auth-Token": api_key}
        self.client = httpx.AsyncClient(
            base_url=server_url, 
            headers=self.headers, 
            timeout=10.0,
            verify=verify_ssl
        )

    async def get_feeds(self) -> List[Feed]:
        # Fetch all feeds to get their titles and IDs
        response = await self.client.get("/v1/feeds")
        response.raise_for_status()
        feeds_data = response.json()
        
        feeds_dict = {f["id"]: Feed(id=f["id"], title=f["title"], unread_count=0) for f in feeds_data}
        
        # Fetch unread counts by fetching entries with status=unread.
        # Miniflux API doesn't easily provide unread count per feed in the /v1/feeds endpoint.
        # We fetch unread entries to aggregate. To ensure we get ALL, we might need to loop or use a high limit.
        # A better way is to use the /v1/feeds and check if there's an unread count there (standard Miniflux doesn't).
        # However, we can fetch entries with a large limit or paginate.
        
        offset = 0
        limit = 1000
        while True:
            entries_response = await self.client.get("/v1/entries", params={"status": "unread", "limit": limit, "offset": offset})
            entries_response.raise_for_status()
            data = entries_response.json()
            entries_data = data["entries"]
            total = data.get("total", 0)
            
            for entry in entries_data:
                f_id = entry["feed_id"]
                if f_id in feeds_dict:
                    feeds_dict[f_id].unread_count += 1
            
            offset += limit
            if offset >= total or not entries_data:
                break
        
        return [f for f in feeds_dict.values() if f.unread_count > 0]

    async def get_entries(self, feed_id: Optional[int] = None, status: str = "unread", 
                            offset: int = 0, limit: int = 45) -> List[Entry]:
            """获取条目"""
            if feed_id:
                # 特定feed的条目 ✅ 使用正确的端点
                response = await self.client.get(
                    f"/v1/feeds/{feed_id}/entries",
                    params={"status": status, "limit": limit, "offset": offset}
                )
            else:
                # 所有条目
                response = await self.client.get(
                    "/v1/entries",
                    params={"status": status, "limit": limit, "offset": offset}
                )
            
            response.raise_for_status()
            data = response.json()
            
            entries = []
            for e in data["entries"]:
                entries.append(Entry(
                    id=e["id"],
                    title=e["title"],
                    url=e["url"],
                    content=e["content"],
                    status=e["status"],
                    starred=e["starred"],
                    feed_id=e["feed_id"],
                    feed_title=e["feed"]["title"] if e.get("feed") else "",
                    published_at=e["published_at"]
                ))
            return entries

    async def update_entries_status(self, entry_ids: List[int], status: str):
        response = await self.client.put("/v1/entries", json={
            "entry_ids": entry_ids,
            "status": status
        })
        response.raise_for_status()

    async def toggle_starred(self, entry_id: int):
        # Miniflux uses PUT /v1/entries/{entry_id}/bookmark to toggle
        response = await self.client.put(f"/v1/entries/{entry_id}/bookmark")
        response.raise_for_status()

    async def get_counters(self) -> Dict[str, int]:
        response = await self.client.get("/v1/feeds/counters")  # ✅ 修正端点
        response.raise_for_status()
        data = response.json()
        # ✅ 修正：返回 unreads 对象，而不是 feeds
        unreads = data.get("unreads", {})
        if unreads:
            return unreads  # 返回 {"feed_id": unread_count}
        
        # Fallback保持不变
        counters = {}
        offset = 0
        limit = 1000
        while True:
            resp = await self.client.get("/v1/entries", params={"status": "unread", "limit": limit, "offset": offset})
            resp.raise_for_status()
            d = resp.json()
            entries = d.get("entries", [])
            total = d.get("total", 0)
            for e in entries:
                fid = str(e.get("feed_id"))
                counters[fid] = counters.get(fid, 0) + 1
            offset += limit
            if offset >= total or not entries:
                break
        return counters

    async def get_read_entries_count(self) -> int:
        resp = await self.client.get("/v1/entries", params={"status": "read", "limit": 1})
        resp.raise_for_status()
        return resp.json().get("total", 0)

    async def flush_history(self) -> None:
        resp = await self.client.put("/v1/flush-history")
        resp.raise_for_status()

    async def close(self):
        await self.client.aclose()
