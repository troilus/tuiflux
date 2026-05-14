package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/anomalyco/tuiflux/internal/api"
	"github.com/anomalyco/tuiflux/internal/config"
	"github.com/anomalyco/tuiflux/internal/locale"
	"github.com/anomalyco/tuiflux/internal/models"
)

const (
	Version  = "0.7"
	PageSize = 15
)

type App struct {
	*tview.Application
	pages     *tview.Pages
	feedList  *tview.List
	entryList *tview.Table
	preview   *tview.TextView

	feedFlex       *tview.Flex
	mainFlex       *tview.Flex
	feedsLabel     *tview.TextView
	entriesLabel   *tview.TextView

	api *api.Client
	cfg *config.Config
	L   locale.Locale

	mu             sync.Mutex
	allFeedsData   map[int]*models.Feed
	feedsOrder     []int
	entries        []models.Entry
	currentFeedID  int
	exhaustedFeeds map[int]bool
	entryPage      int
	lastFocusID    string
}

func openURL(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}

func relativeTime(pubAt string) string {
	t, err := time.Parse(time.RFC3339, pubAt)
	if err != nil {
		return ""
	}
	hours := int(time.Since(t).Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", hours/24)
}

func New(cfg *config.Config) *App {
	a := &App{
		Application:    tview.NewApplication(),
		allFeedsData:   make(map[int]*models.Feed),
		exhaustedFeeds: make(map[int]bool),
		cfg:            cfg,
		L:              locale.Get(cfg.Language),
		api:            api.New(cfg.ServerURL, cfg.APIKey, cfg.VerifySSL),
	}

	a.feedList = tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			a.selectFeed(index)
		})

	a.entryList = tview.NewTable().
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorOrange)).
		SetSelectionChangedFunc(func(row, column int) {
			a.onEntryHighlight(row)
		})
	a.entryList.SetFixed(1, 0)

	a.preview = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetText(a.L["select_entry_preview"])
	a.preview.SetBorder(true)

	a.feedsLabel = tview.NewTextView().
		SetText(a.L["feeds"]).
		SetTextAlign(tview.AlignCenter)

	a.entriesLabel = tview.NewTextView().
		SetText(a.L["entries"]).
		SetTextAlign(tview.AlignCenter)

	entryFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.entriesLabel, 1, 0, false).
		AddItem(a.entryList, 0, 1, false)

	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(entryFlex, 0, 2, false).
		AddItem(a.preview, 0, 3, false)

	leftPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.feedsLabel, 1, 0, false).
		AddItem(a.feedList, 0, 1, false)

	a.mainFlex = tview.NewFlex().
		AddItem(leftPane, 0, 3, true).
		AddItem(rightPane, 0, 7, false)

	a.pages = tview.NewPages().
		AddPage("main", a.mainFlex, true, true)

	a.SetRoot(a.pages, true).
		SetFocus(a.feedList).
		SetInputCapture(a.globalInputCapture)

	a.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		go a.initialLoad()
		a.SetBeforeDrawFunc(nil)
		return false
	})

	return a
}

func (a *App) globalInputCapture(event *tcell.EventKey) *tcell.EventKey {
	page, _ := a.pages.GetFrontPage()
	if page != "main" {
		return event
	}
	switch event.Rune() {
	case 'q':
		a.Stop()
		return nil
	case 'f':
		go a.initialLoad()
		return nil
	case '?':
		a.ShowHelp()
		return nil
	case 'S':
		a.ShowSettings()
		return nil
	case 'H':
		go a.FlushHistory()
		return nil
	case '\t':
		if a.GetFocus() == a.feedList {
			a.SetFocus(a.entryList)
		} else {
			a.SetFocus(a.feedList)
		}
		return nil
	}
	return event
}

func (a *App) entryListInputCapture(event *tcell.EventKey) *tcell.EventKey {
	page, _ := a.pages.GetFrontPage()
	if page != "main" {
		return event
	}
	switch event.Rune() {
	case 'm':
		go a.toggleSelectedRead()
		return nil
	case 's':
		go a.toggleSelectedStar()
		return nil
	case 'o':
		a.openSelectedInBrowser()
		return nil
	case 'r':
		go a.markPageRead()
		return nil
	case ' ':
		go a.readAndNext()
		return nil
	}
	switch event.Key() {
	case tcell.KeyEnter:
		a.openSelectedReader()
		return nil
	case tcell.KeyInsert:
		a.prevFeed()
		return nil
	case tcell.KeyDelete:
		a.nextFeed()
		return nil
	case tcell.KeyPgUp:
		a.pageUp()
		return nil
	case tcell.KeyPgDn:
		a.pageDown()
		return nil
	}
	return event
}

func (a *App) feedListInputCapture(event *tcell.EventKey) *tcell.EventKey {
	page, _ := a.pages.GetFrontPage()
	if page != "main" {
		return event
	}
	switch event.Key() {
	case tcell.KeyInsert:
		a.prevFeed()
		return nil
	case tcell.KeyDelete:
		a.nextFeed()
		return nil
	}
	return event
}

func (a *App) selectFeed(index int) {
	if index < 0 || index >= len(a.feedsOrder) {
		return
	}
	fid := a.feedsOrder[index]
	if fid == a.currentFeedID {
		a.SetFocus(a.entryList)
		return
	}
	a.currentFeedID = fid
	a.entryPage = 0
	a.entries = nil
	a.entryList.Clear()
	go a.loadEntries()
	a.SetFocus(a.entryList)
}

func (a *App) initialLoad() {
	a.QueueUpdateDraw(func() {
		a.preview.SetText(a.L["fetching_feeds"])
	})

	feeds, err := a.api.GetFeeds()
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(fmt.Sprintf("Network Error: %v", err))
		})
		return
	}
	if len(feeds) == 0 {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(a.L["no_feeds_found"])
		})
		return
	}

	a.mu.Lock()
	a.allFeedsData = make(map[int]*models.Feed)
	a.feedsOrder = nil
	for _, f := range feeds {
		fCopy := f
		a.allFeedsData[f.ID] = &fCopy
		a.feedsOrder = append(a.feedsOrder, f.ID)
	}
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.rebuildFeedList()
		a.feedList.SetInputCapture(a.feedListInputCapture)
		if len(a.feedsOrder) > 0 {
			if a.currentFeedID == 0 {
				a.currentFeedID = a.feedsOrder[0]
			}
			a.feedList.SetCurrentItem(a.findFeedIndex(a.currentFeedID))
		}
		a.preview.SetText(a.L["select_entry_preview"])
	})

	if a.currentFeedID > 0 {
		a.loadEntries()
	}
}

func (a *App) findFeedIndex(feedID int) int {
	for i, id := range a.feedsOrder {
		if id == feedID {
			return i
		}
	}
	return 0
}

func (a *App) rebuildFeedList() {
	a.feedList.Clear()

	totalUnread := 0
	for _, f := range a.allFeedsData {
		totalUnread += f.UnreadCount
	}
	a.feedsLabel.SetText(fmt.Sprintf("%s (%d)", a.L["feeds"], totalUnread))

	for _, id := range a.feedsOrder {
		if f, ok := a.allFeedsData[id]; ok && f.UnreadCount > 0 {
			text := fmt.Sprintf("%s (%d)", f.Title, f.UnreadCount)
			a.feedList.AddItem(text, "", 0, nil)
		}
	}
}

func (a *App) loadEntries() {
	a.mu.Lock()
	fid := a.currentFeedID
	a.mu.Unlock()

	if fid == 0 {
		return
	}

	a.QueueUpdateDraw(func() {
		a.entriesLabel.SetText(fmt.Sprintf("%s... (%s...)", a.L["entries_of"], a.L["loading"]))
	})

	entries, err := a.api.GetEntries(fid, "unread", 0, 45)
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(fmt.Sprintf("Error: %v", err))
		})
		return
	}

	a.mu.Lock()
	a.entries = entries
	a.entryPage = 0
	delete(a.exhaustedFeeds, fid)
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
	})
}

func (a *App) fetchMoreEntries() {
	a.mu.Lock()
	fid := a.currentFeedID
	exhausted := a.exhaustedFeeds[fid]
	offset := len(a.entries)
	a.mu.Unlock()

	if fid == 0 || exhausted {
		return
	}

	newEntries, err := a.api.GetEntries(fid, "unread", offset, 45)
	if err != nil {
		return
	}

	if len(newEntries) == 0 {
		a.mu.Lock()
		a.exhaustedFeeds[fid] = true
		a.mu.Unlock()
		return
	}

	a.mu.Lock()
	a.entries = append(a.entries, newEntries...)
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
	})
}

func (a *App) renderEntryPage() {
	a.mu.Lock()
	entries := a.entries
	page := a.entryPage
	fid := a.currentFeedID
	a.mu.Unlock()

	totalPages := (len(entries) + PageSize - 1) / PageSize
	if totalPages == 0 {
		totalPages = 1
	}
	currentPage := page + 1
	if len(entries) == 0 {
		currentPage = 0
	}

	feedName := ""
	if f, ok := a.allFeedsData[fid]; ok {
		feedName = f.Title
	}
	a.entriesLabel.SetText(fmt.Sprintf("%s %s (%d/%d)", a.L["entries_of"], feedName, currentPage, totalPages))

	a.entryList.Clear()
	start := page * PageSize
	end := start + PageSize
	if end > len(entries) {
		end = len(entries)
	}
	pageEntries := entries[start:end]

	for i, entry := range pageEntries {
		starChar := " "
		if entry.Starred {
			starChar = "★"
		}
		timeStr := relativeTime(entry.PublishedAt)
		title := fmt.Sprintf("%s %s", starChar, entry.Title)

		color := tcell.ColorWhite
		if entry.Status == "read" {
			color = tcell.ColorGray
		}
		if entry.Starred {
			color = tcell.ColorOrangeRed
		}

		cell0 := tview.NewTableCell(title).
			SetTextColor(color).
			SetMaxWidth(0).
			SetExpansion(1)
		cell1 := tview.NewTableCell(timeStr).
			SetTextColor(color).
			SetAlign(tview.AlignRight).
			SetMaxWidth(6)

		a.entryList.SetCell(i, 0, cell0)
		a.entryList.SetCell(i, 1, cell1)
	}

	a.entryList.SetInputCapture(a.entryListInputCapture)

	if currentPage == totalPages && len(entries) > 0 {
		go a.fetchMoreEntries()
	}
}

func (a *App) onEntryHighlight(row int) {
	a.mu.Lock()
	entries := a.entries
	page := a.entryPage
	a.mu.Unlock()

	idx := page*PageSize + row
	if idx >= 0 && idx < len(entries) {
		e := entries[idx]
		timeStr := e.PublishedAt
		if len(timeStr) > 10 {
			timeStr = timeStr[:10]
		}
		content := htmlToMarkdown(e.Content)
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		previewText := fmt.Sprintf("%s: %s\n%s: %s\n\n%s",
			a.L["time"], timeStr,
			a.L["source"], e.URL,
			content)
		a.preview.SetText(previewText)
	}
}

func (a *App) getSelectedEntryIdx() int {
	row, _ := a.entryList.GetSelection()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.entryPage*PageSize + row
}

func (a *App) getSelectedEntry() *models.Entry {
	idx := a.getSelectedEntryIdx()
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx >= 0 && idx < len(a.entries) {
		return &a.entries[idx]
	}
	return nil
}

func (a *App) toggleSelectedRead() {
	entry := a.getSelectedEntry()
	if entry == nil {
		return
	}
	a.toggleReadEntry(entry)
}

func (a *App) toggleReadEntry(entry *models.Entry) {
	newStatus := "read"
	if entry.Status == "read" {
		newStatus = "unread"
	}
	if err := a.api.UpdateEntriesStatus([]int{entry.ID}, newStatus); err != nil {
		return
	}
	entry.Status = newStatus

	a.mu.Lock()
	a.syncFeedCount(entry.FeedID)
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
		a.rebuildFeedList()
	})
}

func (a *App) toggleSelectedStar() {
	entry := a.getSelectedEntry()
	if entry == nil {
		return
	}
	a.toggleStarEntry(entry)
}

func (a *App) toggleStarEntry(entry *models.Entry) {
	if err := a.api.ToggleStarred(entry.ID); err != nil {
		return
	}
	entry.Starred = !entry.Starred

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
	})
}

func (a *App) openSelectedInBrowser() {
	entry := a.getSelectedEntry()
	if entry == nil {
		return
	}
	openURL(entry.URL)
}

func (a *App) openSelectedReader() {
	entry := a.getSelectedEntry()
	if entry == nil {
		return
	}

	if entry.Status == "unread" {
		go func() {
			if err := a.api.UpdateEntriesStatus([]int{entry.ID}, "read"); err == nil {
				entry.Status = "read"
				a.mu.Lock()
				a.syncFeedCount(entry.FeedID)
				a.mu.Unlock()
				a.QueueUpdateDraw(func() {
					a.renderEntryPage()
					a.rebuildFeedList()
				})
			}
			a.QueueUpdateDraw(func() {
				showReader(a, entry)
			})
		}()
		return
	}

	a.QueueUpdateDraw(func() {
		showReader(a, entry)
	})
}

func (a *App) readAndNext() {
	entry := a.getSelectedEntry()
	if entry == nil {
		return
	}

	if err := a.api.UpdateEntriesStatus([]int{entry.ID}, "read"); err != nil {
		return
	}
	entry.Status = "read"

	a.mu.Lock()
	a.syncFeedCount(entry.FeedID)
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
		a.rebuildFeedList()
	})

	a.mu.Lock()
	row, _ := a.entryList.GetSelection()
	total := len(a.entries)
	page := a.entryPage
	nextIdx := page*PageSize + row + 1
	a.mu.Unlock()

	if nextIdx < total {
		a.QueueUpdateDraw(func() {
			nextRow := row + 1
			if nextRow >= PageSize {
				a.entryPage++
				a.renderEntryPage()
				nextRow = 0
			}
			if nextRow < a.entryList.GetRowCount() {
				a.entryList.Select(nextRow, 0)
			}
		})
	} else {
		a.jumpToNextFeed()
	}
}

func (a *App) markPageRead() {
	a.mu.Lock()
	entries := a.entries
	page := a.entryPage
	a.mu.Unlock()

	start := page * PageSize
	end := start + PageSize
	if end > len(entries) {
		end = len(entries)
	}

	var toMark []int
	for i := start; i < end; i++ {
		if entries[i].Status == "unread" {
			toMark = append(toMark, entries[i].ID)
		}
	}

	if len(toMark) == 0 {
		return
	}

	if err := a.api.UpdateEntriesStatus(toMark, "read"); err != nil {
		return
	}

	a.mu.Lock()
	for i := start; i < end; i++ {
		if entries[i].Status == "unread" {
			entries[i].Status = "read"
		}
		a.syncFeedCount(entries[i].FeedID)
	}
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
		a.rebuildFeedList()
	})

	a.mu.Lock()
	nextStart := (page + 1) * PageSize
	if nextStart < len(a.entries) {
		a.entryPage++
		a.mu.Unlock()
		a.QueueUpdateDraw(func() {
			a.renderEntryPage()
			a.entryList.Select(0, 0)
		})
	} else {
		a.mu.Unlock()
		a.jumpToNextFeed()
	}
}

func (a *App) syncFeedCount(feedID int) {
	if f, ok := a.allFeedsData[feedID]; ok {
		count := 0
		for _, e := range a.entries {
			if e.FeedID == feedID && e.Status == "unread" {
				count++
			}
		}
		f.UnreadCount = count
	}
}

func (a *App) pageUp() {
	a.mu.Lock()
	if a.entryPage > 0 {
		a.entryPage--
	}
	a.mu.Unlock()
	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
	})
}

func (a *App) pageDown() {
	a.mu.Lock()
	totalPages := (len(a.entries) + PageSize - 1) / PageSize
	if a.entryPage < totalPages-1 {
		a.entryPage++
	} else {
		a.mu.Unlock()
		go a.fetchMoreEntries()
		return
	}
	a.mu.Unlock()
	a.QueueUpdateDraw(func() {
		a.renderEntryPage()
	})
}

func (a *App) prevFeed() {
	a.mu.Lock()
	order := a.feedsOrder
	cur := a.currentFeedID
	a.mu.Unlock()

	if len(order) == 0 {
		return
	}
	idx := a.findFeedIndex(cur)
	if idx > 0 {
		idx--
	}
	a.mu.Lock()
	a.currentFeedID = order[idx]
	a.entryPage = 0
	a.entries = nil
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.entryList.Clear()
		a.feedList.SetCurrentItem(idx)
	})
	go a.loadEntries()
}

func (a *App) nextFeed() {
	a.mu.Lock()
	order := a.feedsOrder
	cur := a.currentFeedID
	a.mu.Unlock()

	if len(order) == 0 {
		return
	}
	idx := a.findFeedIndex(cur)
	if idx < len(order)-1 {
		idx++
	}
	a.mu.Lock()
	a.currentFeedID = order[idx]
	a.entryPage = 0
	a.entries = nil
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.entryList.Clear()
		a.feedList.SetCurrentItem(idx)
	})
	go a.loadEntries()
}

func (a *App) jumpToNextFeed() {
	a.mu.Lock()
	order := a.feedsOrder
	cur := a.currentFeedID
	a.mu.Unlock()

	if len(order) == 0 {
		return
	}

	idx := a.findFeedIndex(cur)
	if idx < len(order)-1 {
		idx++
	}
	a.mu.Lock()
	a.currentFeedID = order[idx]
	a.entryPage = 0
	a.entries = nil
	a.mu.Unlock()

	a.QueueUpdateDraw(func() {
		a.entryList.Clear()
		a.feedList.SetCurrentItem(idx)
		a.SetFocus(a.feedList)
	})
	go a.loadEntries()
	go func() {
		time.Sleep(100 * time.Millisecond)
		a.QueueUpdateDraw(func() {
			a.SetFocus(a.entryList)
			if a.entryList.GetRowCount() > 0 {
				a.entryList.Select(0, 0)
			}
		})
	}()
}

func (a *App) FlushHistory() {
	count, err := a.api.GetReadEntriesCount()
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(fmt.Sprintf("Error: %v", err))
		})
		return
	}
	if count == 0 {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(a.L["flush_no_entries"])
		})
		return
	}

	msg := fmt.Sprintf(a.L["flush_confirm"], count)
	a.QueueUpdateDraw(func() {
		showConfirm(a, msg, func(confirmed bool) {
			if confirmed {
				go a.executeFlush()
			}
		})
	})
}

func (a *App) executeFlush() {
	if err := a.api.FlushHistory(); err != nil {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(fmt.Sprintf("Error: %v", err))
		})
		return
	}
	a.mu.Lock()
	a.currentFeedID = 0
	a.entries = nil
	a.exhaustedFeeds = make(map[int]bool)
	a.mu.Unlock()
	a.QueueUpdateDraw(func() {
		a.entryList.Clear()
		a.preview.SetText(a.L["flush_success"])
	})
	go a.initialLoad()
}

func (a *App) ShowSettings() {
	a.QueueUpdateDraw(func() {
		showSettings(a)
	})
}

func (a *App) ShowHelp() {
	a.QueueUpdateDraw(func() {
		showHelp(a)
	})
}


