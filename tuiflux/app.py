import webbrowser
import asyncio
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

class ReaderScreen(Screen):
    BINDINGS = [
        Binding("escape", "app.pop_screen", "Back to List"),
        Binding("shift+j", "open_in_browser", "Open in Browser"),
        Binding("m", "toggle_read", "Toggle Read/Unread"),
        Binding("up", "scroll_up", "Scroll Up", show=False),
        Binding("down", "scroll_down", "Scroll Down", show=False),
        Binding("pageup", "page_up", "Page Up"),
        Binding("pagedown", "page_down", "Page Down"),
    ]

    def __init__(self, entry: Entry, app_ref):
        super().__init__()
        self.entry = entry
        self.app_ref = app_ref

    def compose(self) -> ComposeResult:
        yield Header()
        yield Label(f"Status: {self.entry.status.upper()}", id="reader-status")
        with ScrollableContainer(id="reader-container"):
            yield Static(f"# {self.entry.title}\n\n", id="reader-title")
            yield Static(Markdown(self.entry.content))
        yield Footer()

    def action_open_in_browser(self):
        webbrowser.open(self.entry.url)

    async def action_toggle_read(self):
        new_status = "read" if self.entry.status == "unread" else "unread"
        await self.app_ref.api.update_entries_status([self.entry.id], new_status)
        self.entry.status = new_status
        self.query_one("#reader-status", Label).update(f"Status: {self.entry.status.upper()}")
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
        yield Label(self.get_label_text(), id="entry-label")

    def get_label_text(self):
        star = "★ " if self.entry.starred else "  "
        return f"{star}{self.entry.title}"

    def update_style(self):
        label = self.query_one("#entry-label", Label)
        label.update(self.get_label_text())
        if self.entry.starred:
            label.styles.color = "gold"
        elif self.entry.status == "read":
            label.styles.color = "gray"
        else:
            label.styles.color = "white"

    def on_mount(self):
        self.update_style()

class TuifluxApp(App):

    CSS = """
    #left-pane {
        width: 30%;
        border-right: solid green;
    }
    #right-pane {
        width: 70%;
    }
    #entry-list-container {
        height: 17;
        border-bottom: solid green;
    }
    #preview-pane {
        height: 1fr;
        padding: 1 2;
    }
    #preview-url {
        color: blue;
        text-style: underline;
        margin-top: 1;
    }
    ListView {
        scrollbar-size: 0 0;
        height: 100%;
        border: none;
    }
    ListView:focus {
        border: none;
    }
    ListItem {
        padding: 0 1;
        background: transparent;
    }
    ListItem:focus {
        background: $accent;
        color: $text;
        text-style: bold;
    }
    ListItem:focus Label {
        color: $text;
        text-style: bold;
    }
    ListItem.--highlight {
        background: $accent-darken-1;
    }
    ListItem.--highlight Label {
        text-style: bold;
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
        Binding("tab", "switch_focus", "Switch Pane"),
        Binding("m", "toggle_read", "Read/Unread"),
        Binding("space", "read_and_next", "Read and Next"),
        Binding("shift+s", "toggle_star", "Star/Unstar"),
        Binding("shift+r", "mark_page_read", "Mark Page Read"),
        Binding("enter", "handle_enter", "Enter"),
        Binding("pageup", "page_up", "Page Up"),
        Binding("pagedown", "page_down", "Page Down"),
        Binding("q", "quit", "Quit"),
    ]

    entry_page = reactive(0)
    PAGE_SIZE = 15

    def __init__(self):
        super().__init__()
        config = load_config()
        self.api = MinifluxAPI(config["server_url"], config["api_key"], verify_ssl=config.get("verify_ssl", True))
        self.all_feeds_data = {} 
        self.entries = []
        self.current_feed_id = None

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
        try:
            overlay.update("Fetching feeds and counts...")
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
            
            self.all_feeds_data = {}
            for f in feeds_data:
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
        await feed_list.clear()
        # Only show feeds with unread items as requested
        for f in self.all_feeds_data.values():
            if f.unread_count > 0:
                await feed_list.append(FeedItem(f))

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
        self.query_one("#preview-content", Static).update(Markdown(entry.content[:500] + "..."))
        self.query_one("#preview-url", Static).update(f"Source: {entry.url}")

    async def load_entries(self, feed_id: int):
        self.entries = await self.api.get_entries(feed_id)
        await self.refresh_entry_list()

    async def refresh_entry_list(self):
        entry_list = self.query_one("#entry-list", ListView)
        await entry_list.clear()
        start = self.entry_page * self.PAGE_SIZE
        page_entries = self.entries[start:start + self.PAGE_SIZE]
        for entry in page_entries:
            await entry_list.append(EntryItem(entry))
        if entry_list.children:
            entry_list.index = 0
            self.update_preview(entry_list.children[0].entry)

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
                if item.entry.status == "unread":
                    await self.api.update_entries_status([item.entry.id], "read")
                    item.entry.status = "read"
                    item.update_style()
                    await self.sync_feed_count(item.entry.feed_id, "unread", "read")
                
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
