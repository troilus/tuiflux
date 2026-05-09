# TuiFlux

TuiFlux 是一个基于终端（TUI）的 Miniflux RSS 阅读器客户端，使用 Python 和 Textual 库构建，旨在提供高效、简洁的 RSS 阅读体验。

## 项目介绍

TuiFlux 允许你在终端中直接浏览你的 RSS 订阅源（基于 Miniflux），支持标记已读/未读、收藏/取消收藏、文章预览及在浏览器中打开等功能。

## 安装与运行

### 前置要求

- Python 3.8+
- 一个可访问的 Miniflux 服务器

### 安装步骤

1. 克隆项目到本地：
   ```bash
   git clone https://github.com/yourusername/tuiflux.git
   cd tuiflux
   ```

2. 安装依赖：
   ```bash
   pip install -r requirements.txt
   ```

3. 运行：
   ```bash
   python main.py
   ```
   首次运行将需要你提供Minuflux服务器地址、API token。

## 快捷键

### 全局 / 列表模式
- `q`: 退出程序
- `Tab`: 切换焦点（订阅源列表 ↔ 文章列表）
- `Enter`: 打开选中的文章阅读
- `m`: 标记选中文章为已读/未读
- `Space`: 标记当前文章为已读并跳转至下一条
- `s`: 收藏/取消收藏选中文章
- `o`: 在默认浏览器中打开文章链接
- `r`: 标记当前页面显示的15项文章为已读
- `Insert`: 上一个源
- `Delete`: 下一个源
- `PageUp`/`PageDown`: 文章列表翻页

### 文章阅读模式 (Reader Screen)
- `Escape`: 返回列表
- `m`: 切换已读/未读状态
- `s`: 收藏/取消收藏
- `o`: 在浏览器中打开
- `Up`/`Down`/`PageUp`/`PageDown`: 阅读时滚动内容
