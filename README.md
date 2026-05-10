# TuiFlux

中文介绍 https://github.com/troilus/tuiflux/blob/main/README_zh.md

TuiFlux is a Terminal-based User Interface (TUI) RSS reader client for Miniflux, built with Python and the Textual library. It aims to provide an efficient and minimalist RSS reading experience.

## Introduction

TuiFlux allows you to browse your RSS feeds (powered by Miniflux) directly in your terminal. It supports marking articles as read/unread, bookmarking, previewing articles, and opening them in your browser.

<img width="1113" height="626" alt="图片" src="https://github.com/user-attachments/assets/b56c95ca-d7cc-4754-9354-d1372ef64d4a" />

<img width="1113" height="626" alt="图片" src="https://github.com/user-attachments/assets/91da5939-f122-4b05-8dba-f47b9f13ff74" />



## Installation and Setup
Download form https://github.com/troilus/tuiflux/releases or：
### Prerequisites

- Python 3.8+
- An accessible Miniflux server

### Installation Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/troilus/tuiflux.git
   cd tuiflux
   ```

2. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

3. Run the application:
   ```bash
   python main.py
   ```
   Upon the first launch, you will be prompted to provide your Miniflux server URL and API token.

## Key Bindings

### Global / List Mode
- `q`: Quit the application
- `Tab`: Switch focus (Feeds List ↔ Article List)
- `Enter`: Open selected article
- `m`: Toggle read/unread status
- `Space`: Mark current article as read and move to the next
- `s`: Bookmark/Unbookmark selected article
- `o`: Open article link in default browser
- `r`: Mark the visible 15 articles as read
- `Insert`: Previous Feed
- `Delete`: Next Feed
- `PageUp`/`PageDown`: Scroll article list

### Reader Screen
- `Escape`: Return to list
- `m`: Toggle read/unread status
- `s`: Bookmark/Unbookmark
- `o`: Open in browser
- `Up`/`Down`/`PageUp`/`PageDown`: Scroll content
