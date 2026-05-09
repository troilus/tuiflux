from dataclasses import dataclass
from typing import List, Optional

@dataclass
class Feed:
    id: int
    title: str
    unread_count: int = 0

@dataclass
class Entry:
    id: int
    title: str
    url: str
    content: str
    status: str
    starred: bool
    feed_id: int
    feed_title: str
