import webbrowser
import asyncio
import re
import html
from textual.app import App, ComposeResult
from textual.widgets import Header, Footer, ListView, ListItem, Label, Static
from textual.containers import Horizontal, Vertical, ScrollableContainer
from textual.binding import Binding
from textual.screen import Screen
from textual.reactive import reactive
from rich.markdown import Markdown
from .api import MinifluxAPI
from .models import Feed, Entry
from .config import load_config

def html_to_markdown(html_content: str) -> str:
    """A basic HTML to text/markdown converter using standard library."""
    text = html.unescape(html_content)
    # Basic tag removal and formatting
    text = re.sub(r'</p>', '\n\n', text)
    text = re.sub(r'<br\s*/?>', '\n', text)
    text = re.sub(r'<h[1-6]>(.*?)</h[1-6]>', r'\n# \1\n', text)
    text = re.sub(r'<li>(.*?)</li>', r'- \1\n', text)
    text = re.sub(r'<[^>]+>', '', text)
    return text.strip()

class ReaderScreen(Screen):
    BINDINGS = [
        Binding("escape", "app.pop_screen", "Back to list"),
        Binding("m", "toggle_read", "Read/Unread"),
        Binding("o", "open_in_browser", "Open in Browser"),
        Binding("s", "toggle_star", "Star/Unstar"),
        Binding("up", "scroll_up", "Scroll Up", show=False),
        Binding("down", "scroll_down", "Scroll Down", show=False),
        Binding("pageup", "page_up", "Page Up", show=False),
        Binding("pagedown", "page_down", "Page Down", show=False),
        # 隐藏 App 层面的快捷键，改为继承或自定义实现
        Binding("space", "nothing", "Read and next", show=False),
        Binding("r", "nothing", "List Read", show=False),
        Binding("enter", "nothing", "Read more", show=False),
        Binding("insert", "prev_feed", "Previous Feed", show=False),
        Binding("delete", "next_feed", "Next Feed", show=False),
        Binding("q", "nothing", "", show=False),  # 完全隐藏退出
    ]
    
    

    def __init__(self, entry: Entry, app_ref):
        super().__init__()
        self.entry = entry
        self.app_ref = app_ref

    def compose(self) -> ComposeResult:
        yield Header()
        star_status = "STARRED" if self.entry.starred else "UNSTARRED"
        yield Label(f"Status: {self.entry.status.upper()} | {star_status}", id="reader-status")
        with ScrollableContainer(id="reader-container"):
            yield Static(f"# {self.entry.title}\n\n[Source: {self.entry.url}]\n\n{html_to_markdown(self.entry.content)}", id="reader-content")
        yield Footer()

    def action_none(self):
        pass

    def action_open_in_browser(self):
        webbrowser.open(self.entry.url)

    async def action_toggle_star(self):
        await self.app_ref.api.toggle_starred(self.entry.id)
        self.entry.starred = not self.entry.starred
        star_status = "STARRED" if self.entry.starred else "UNSTARRED"
        self.query_one("#reader-status", Label).update(f"Status: {self.entry.status.upper()} | {star_status}")
        await self.app_ref.update_entry_ui_state(self.entry)

    async def action_toggle_read(self):
        new_status = "read" if self.entry.status == "unread" else "unread"
        await self.app_ref.api.update_entries_status([self.entry.id], new_status)
        self.entry.status = new_status
        star_status = "STARRED" if self.entry.starred else "UNSTARRED"
        self.query_one("#reader-status", Label).update(f"Status: {self.entry.status.upper()} | {star_status}")
        await self.app_ref.update_entry_ui_state(self.entry)

    def action_scroll_up(self):
        self.query_one("#reader-container").scroll_relative(y=-1)

    def action_scroll_down(self):
        self.query_one("#reader-container").scroll_relative(y=1)

    def action_page_up(self):
        self.query_one("#reader-container").scroll_page_up()

    def action_page_down(self):
        self.query_one("#reader-container").scroll_page_down()

class FeedItem(ListItem):
    def __init__(self, feed: Feed):
        super().__init__()
        self.feed = feed

    def compose(self) -> ComposeResult:
        yield Label(f"{self.feed.title} ({self.feed.unread_count})")

class EntryItem(ListItem):
    def __init__(self, entry: Entry):
        super().__init__()
        self.entry = entry

    def compose(self) -> ComposeResult:
        with Horizontal(id="entry-row"):
            star = "★ " if self.entry.starred else "  "
            yield Label(star, id="entry-star")
            yield Label(self.entry.title, id="entry-title")
            yield Label(self.get_time_str(), id="entry-time")

    def get_time_str(self):
        from datetime import datetime, timezone
        try:
            pub_date = datetime.fromisoformat(self.entry.published_at.replace("Z", "+00:00"))
            now = datetime.now(timezone.utc)
            delta = now - pub_date
            hours = int(delta.total_seconds() / 3600)
            return f"{hours}h" if hours < 24 else f"{hours // 24}d"
        except:
            return ""

    def update_style(self):
        star_label = self.query_one("#entry-star", Label)
        title_label = self.query_one("#entry-title", Label)
        time_label = self.query_one("#entry-time", Label)
        star_label.update("★ " if self.entry.starred else "  ")
        title_label.update(self.entry.title)
        time_label.update(self.get_time_str())
        
        color = "gold" if self.entry.starred else ("gray" if self.entry.status == "read" else "white")
        star_label.styles.color = color
        title_label.styles.color = color
        time_label.styles.color = color

    def on_mount(self):
        self.update_style()

class TuifluxApp(App):

    CSS = """
    #left-pane {
        width: 30%;
        border-right: solid orange;
    }
    #right-pane {
        width: 70%;
    }
    #entry-list-container {
        height: 17;
        border-bottom: solid orange;
    }
    #preview-pane {
        height: 1fr;
        padding: 1 2;
    }
    #entry-title {
        width: 1fr;
        overflow: hidden;
    }
    #entry-time {
        width: 5;
        content-align: right middle;
    }
    ListView {
        scrollbar-size: 0 0;
        height: 100%;
        border: none;
    }
    ListView:focus {
        border: none;
    }
    #entry-row {
        height: 1;
        width: 100%;
    }

    
    #reader-container {
        padding: 1 2;
    }
    #reader-status {
        background: $accent;
        color: $text;
        padding: 0 1;
        text-align: right;
    }
    #loading-overlay {
        width: 100%;
        height: 100%;
        content-align: center middle;
        background: $surface;
        color: $text;
        text-style: bold;
    }
    """

    BINDINGS = [
        Binding("m", "toggle_read", "Read/Unread", show=False),
        Binding("space", "read_and_next", "Read/Unread and next"),
        Binding("insert", "prev_feed", "Previous Feed"),
        Binding("delete", "next_feed", "Next Feed"),
        Binding("r", "mark_page_read", "List Read"),
        Binding("o", "open_in_browser", "Open in Browser"),
        Binding("s", "toggle_star", "Star/Unstar"),
        Binding("enter", "handle_enter", "Read more"),
        Binding("pageup", "page_up", "Page Up"),
        Binding("pagedown", "page_down", "Page Down"),

        Binding("q", "quit", "Quit"),
        Binding("tab", "switch_focus", "Switch Pane", show=False),

    ]

    def action_prev_feed(self):
        feed_list = self.query_one("#feed-list", ListView)
        if feed_list.index is not None and feed_list.index > 0:
            feed_list.index -= 1
            self.query_one("#feed-list").focus()

    def action_next_feed(self):
        feed_list = self.query_one("#feed-list", ListView)
        if feed_list.index is not None and feed_list.index < len(feed_list.children) - 1:
            feed_list.index += 1
            self.query_one("#feed-list").focus()

    def action_none(self):
        pass

    entry_page = reactive(0)
    PAGE_SIZE = 15

    def __init__(self):
        super().__init__()
        config = load_config()
        self.api = MinifluxAPI(config["server_url"], config["api_key"], verify_ssl=config.get("verify_ssl", True))
        self.all_feeds_data = {} 
        self.entries = []
        self.current_feed_id = None
        self.exhausted_feeds = set()

    def compose(self) -> ComposeResult:
        yield Header()
        yield Static("Initializing...", id="loading-overlay")
        with Horizontal(id="main-container"):
            with Vertical(id="left-pane"):
                yield Label("Feeds", id="feeds-label")
                yield ListView(id="feed-list")
            with Vertical(id="right-pane"):
                with Vertical(id="entry-list-container"):
                    yield Label("Entries", id="entries-label")
                    yield ListView(id="entry-list")
                with Vertical(id="preview-pane"):
                    yield Static("Select an entry to preview", id="preview-content")
                    yield Static("", id="preview-url")
        yield Footer()

    async def on_mount(self) -> None:
        self.query_one("#main-container").display = False
        self.run_worker(self.initial_load())

    async def initial_load(self):
        overlay = self.query_one("#loading-overlay", Static)
        overlay.update("Fetching feeds and counts...")
        try:
            # Use gather but wrap in a try-except for more specific error info if needed
            try:
                feeds_resp, counters = await asyncio.gather(
                    self.api.client.get("/v1/feeds"),
                    self.api.get_counters()
                )
                feeds_resp.raise_for_status()
            except Exception as e:
                overlay.update(f"Network Error: {e}")
                return

            feeds_data = feeds_resp.json()
            total_feeds = len(feeds_data)
            
            self.all_feeds_data = {}
            for i, f in enumerate(feeds_data, 1):
                overlay.update(f"Fetching feeds and counts... {i}")
                key = f["id"]
                count = counters.get(str(key), counters.get(key, 0))
                self.all_feeds_data[key] = Feed(id=key, title=f["title"], unread_count=count)
            
            if not self.all_feeds_data:
                overlay.update("No feeds found on server.")
                return

            overlay.display = False
            self.query_one("#main-container").display = True
            await self.refresh_feed_list_ui()
            
            feed_list = self.query_one("#feed-list", ListView)
            if feed_list.children:
                feed_list.index = 0
                self.current_feed_id = feed_list.children[0].feed.id
                await self.load_entries(self.current_feed_id)
            feed_list.focus()
            
            self.run_worker(self.background_count_sync())
        except Exception as e:
            overlay.update(f"Initialization Error: {e}")
            self.log(f"Initial load error: {e}")

    async def background_count_sync(self):
        try:
            # We already have counts from counters API, deep sync is optional but kept for completeness
            # Ensuring counts are updated if counters were stale
            pass
        except Exception as e: self.log(f"Background sync error: {e}")

    async def refresh_feed_list_ui(self):
        feed_list = self.query_one("#feed-list", ListView)
        current_index = feed_list.index
        await feed_list.clear()
        total_unread = sum(f.unread_count for f in self.all_feeds_data.values())
        self.query_one("#feeds-label", Label).update(f"Feeds ({total_unread})")
        # Only show feeds with unread items as requested
        for f in self.all_feeds_data.values():
            if f.unread_count > 0:
                await feed_list.append(FeedItem(f))
        
        if current_index is not None and current_index < len(feed_list.children):
            feed_list.index = current_index
        elif feed_list.children:
            feed_list.index = 0

    async def on_list_view_selected(self, event: ListView.Selected) -> None:
        if event.list_view.id == "feed-list":
            self.current_feed_id = event.item.feed.id
            self.entry_page = 0
            await self.load_entries(self.current_feed_id)
            self.query_one("#entry-list").focus()
        elif event.list_view.id == "entry-list":
            await self.action_open_reader()

    async def on_list_view_highlighted(self, event: ListView.Highlighted) -> None:
        if event.list_view.id == "entry-list" and event.item:
            if isinstance(event.item, EntryItem):
                self.update_preview(event.item.entry)

    def update_preview(self, entry: Entry):
        time_str = entry.published_at.split('T')[0] # Using simple date format
        preview_text = f"TIME: {time_str}\nSource: {entry.url}\n\n{html_to_markdown(entry.content[:500] + '...')}"
        self.query_one("#preview-content", Static).update(preview_text)
        self.query_one("#preview-url", Static).update("") # Clear this as it's now in content

    async def load_entries(self, feed_id: int):
        self.entries = await self.api.get_entries(feed_id, offset=0)
        self.entry_page = 0
        await self.refresh_entry_list()

    async def fetch_more_entries(self):
        if self.current_feed_id in self.exhausted_feeds:
            return
            
        self.query_one("#entries-label", Label).update(f"Entries... (loading...)")
        new_entries = await self.api.get_entries(self.current_feed_id, offset=len(self.entries))
        if new_entries:
            self.entries.extend(new_entries)
            await self.refresh_entry_list()
        else:
            self.exhausted_feeds.add(self.current_feed_id)
            await self.refresh_entry_list() # Update label to remove loading...

    async def refresh_entry_list(self):
        entry_list = self.query_one("#entry-list", ListView)
        current_index = entry_list.index
        await entry_list.clear()
        
        feed = self.all_feeds_data.get(self.current_feed_id)
        feed_name = feed.title if feed else "Unknown"
        
        total_pages = (len(self.entries) + self.PAGE_SIZE - 1) // self.PAGE_SIZE
        current_page = self.entry_page + 1 if self.entries else 0
        self.query_one("#entries-label", Label).update(f"Entries of {feed_name} ({current_page}/{total_pages})")
        
        start = self.entry_page * self.PAGE_SIZE
        page_entries = self.entries[start:start + self.PAGE_SIZE]
        for entry in page_entries:
            await entry_list.append(EntryItem(entry))
        
        if current_index is not None and current_index < len(entry_list.children):
            entry_list.index = current_index
        elif entry_list.children:
            entry_list.index = 0
            
        if entry_list.index is not None and entry_list.index < len(entry_list.children):
            self.update_preview(entry_list.children[entry_list.index].entry)
            
        # Trigger pre-fetch if on last page
        if current_page == total_pages:
            self.run_worker(self.fetch_more_entries())

    def check_pre_fetch(self):
        # Trigger pre-fetch if on last page
        total_pages = (len(self.entries) + self.PAGE_SIZE - 1) // self.PAGE_SIZE
        if self.entry_page == total_pages - 1:
            self.run_worker(self.fetch_more_entries())

    def action_open_in_browser(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        if entry_list.index is not None:
            item = entry_list.children[entry_list.index]
            if isinstance(item, EntryItem):
                webbrowser.open(item.entry.url)

    async def action_handle_enter(self):
        # Focus handling for feed-list is in on_list_view_selected
        if self.focused and self.focused.id == "entry-list":
            await self.action_open_reader()

    def action_page_up(self) -> None:
        if self.focused and self.focused.id == "entry-list":
            if self.entry_page > 0:
                self.entry_page -= 1
                self.run_worker(self.refresh_entry_list())

    def action_page_down(self) -> None:
        if self.focused and self.focused.id == "entry-list":
            if (self.entry_page + 1) * self.PAGE_SIZE < len(self.entries):
                self.entry_page += 1
                self.run_worker(self.refresh_entry_list())
            else:
                self.run_worker(self.fetch_more_entries())

    async def action_toggle_read(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        if entry_list.index is not None:
            item = entry_list.children[entry_list.index]
            if isinstance(item, EntryItem):
                old_status = item.entry.status
                new_status = "read" if old_status == "unread" else "unread"
                await self.api.update_entries_status([item.entry.id], new_status)
                item.entry.status = new_status
                item.update_style()
                await self.sync_feed_count(item.entry.feed_id, old_status, new_status)

    async def action_read_and_next(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        if entry_list.index is not None:
            item = entry_list.children[entry_list.index]
            if isinstance(item, EntryItem):
                # Toggle status: unread -> read, read -> unread
                old_status = item.entry.status
                new_status = "read" if old_status == "unread" else "unread"
                
                await self.api.update_entries_status([item.entry.id], new_status)
                item.entry.status = new_status
                item.update_style()
                await self.sync_feed_count(item.entry.feed_id, old_status, new_status)
                
                if entry_list.index < len(entry_list.children) - 1:
                    entry_list.index += 1
                elif (self.entry_page + 1) * self.PAGE_SIZE < len(self.entries):
                    self.entry_page += 1
                    await self.refresh_entry_list()
                    entry_list.index = 0

    async def sync_feed_count(self, feed_id, old_status, new_status):
        feed_data = self.all_feeds_data.get(feed_id)
        if feed_data:
            if old_status == "unread" and new_status == "read":
                feed_data.unread_count = max(0, feed_data.unread_count - 1)
            elif old_status == "read" and new_status == "unread":
                feed_data.unread_count += 1
            await self.refresh_feed_list_ui()

    async def action_mark_page_read(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        to_mark = [item.entry for item in entry_list.children if isinstance(item, EntryItem) and item.entry.status == "unread"]
        if to_mark:
            await self.api.update_entries_status([e.id for e in to_mark], "read")
            for e in to_mark:
                e.status = "read"
                await self.sync_feed_count(e.feed_id, "unread", "read")
            
            if (self.entry_page + 1) * self.PAGE_SIZE < len(self.entries):
                self.entry_page += 1
                await self.refresh_entry_list()
                entry_list.index = 0
            else:
                for item in entry_list.children: item.update_style()

    async def update_entry_ui_state(self, entry: Entry):
        entry_list = self.query_one("#entry-list", ListView)
        for item in entry_list.children:
            if isinstance(item, EntryItem) and item.entry.id == entry.id:
                item.update_style()
                break
        await self.sync_feed_count(entry.feed_id, "unknown", entry.status)

    async def action_open_reader(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        if entry_list.index is not None:
            item = entry_list.children[entry_list.index]
            if isinstance(item, EntryItem):
                if item.entry.status == "unread":
                    await self.api.update_entries_status([item.entry.id], "read")
                    item.entry.status = "read"
                    item.update_style()
                    await self.sync_feed_count(item.entry.feed_id, "unread", "read")
                self.push_screen(ReaderScreen(item.entry, self))

    async def action_toggle_star(self) -> None:
        entry_list = self.query_one("#entry-list", ListView)
        if entry_list.index is not None:
            item = entry_list.children[entry_list.index]
            if isinstance(item, EntryItem):
                await self.api.toggle_starred(item.entry.id)
                item.entry.starred = not item.entry.starred
                item.update_style()

    def action_switch_focus(self) -> None:
        if self.focused and self.focused.id == "feed-list":
            self.query_one("#entry-list").focus()
        else:
            self.query_one("#feed-list").focus()

if __name__ == "__main__":
    TuifluxApp().run()
