---
title: Guppy
---

# Guppy

Guppy is a free, open-source terminal application for macOS and Linux that
lets you view and manage your **Google Tasks** without leaving the keyboard.
It is distributed under the MIT license; the source code lives at
[github.com/kartikssj/guppy](https://github.com/kartikssj/guppy).

## What Guppy does

After you sign in with your Google account, Guppy connects to the Google Tasks
API and syncs with your task lists so you can:

- Browse all your task lists side-by-side with their tasks
- See subtasks rendered as an indented tree
- Add, rename, complete/reopen, and delete tasks and subtasks
- Do all of it entirely from the keyboard — changes sync to Google Tasks
  immediately

```
 ▏                                ▏
 ▏   Lists (2)                    ▏   Personal                       2/4
 ▏                                ▏
 ▏    Personal                    ▏   ☑  Buy milk
 ▏    Work                        ▏   ☐  Write report
 ▏    Shopping                    ▏     └ ☐  Draft outline
 ▏                                ▏   ☐  Call mom

  tab switch · space toggle · a add · A subtask · ? help · q quit
```

## How Guppy uses Google data

- Guppy requests a single OAuth scope — Google Tasks read/write
  (`https://www.googleapis.com/auth/tasks`) — and nothing else.
- Task data travels only between your computer and Google's API servers, and
  is held in memory only while the app runs.
- Guppy stores no task data on disk and contains no analytics, telemetry,
  tracking, or advertising. Its developers operate no servers and receive no
  copies of your data.
- The sign-in token stays on your own device (`~/.config/guppy/token.json`,
  owner-only file permissions) and is used only to authenticate requests to
  Google.

## Install

macOS / Linux (the binary is invoked as `guppy`):

```sh
curl -fsSL https://raw.githubusercontent.com/kartikssj/guppy/main/install.sh | sh
```

Or download a binary from
[Releases](https://github.com/kartikssj/guppy/releases), or build from source.

## Links

- [Source code & documentation](https://github.com/kartikssj/guppy)
- [Privacy Policy](/guppy/privacy/)
- [Terms of Service](/guppy/terms/)
- [Report an issue](https://github.com/kartikssj/guppy/issues)
