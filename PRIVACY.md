# Privacy Policy for guppy

**Effective date:** 2026-09-28
**Contact:** `kartikssj@gmail.com`
**Project:** `https://github.com/kartikssj/guppy`

guppy is a free, open-source terminal application that lets you view and
manage your Google Tasks from the command line. This policy explains what
data guppy accesses, how it is used, and how it is stored.

The short version: **guppy runs entirely on your computer, talks only to
Google, and collects nothing.** The developers of guppy never see, receive,
or store your data.

## Data guppy accesses

With your explicit consent, guppy accesses your Google Tasks data through
the Google Tasks API, using the single scope
`https://www.googleapis.com/auth/tasks` (read and write tasks). This includes:

- Task list names and identifiers
- Task titles, notes, due dates, completion status, and hierarchy

guppy accesses this data only to display it to you and to perform the actions
you request (create, complete, rename, delete tasks). It requests the minimum
scope required for this functionality and nothing else.

## How your data is used

- Task data is fetched from Google, rendered in your terminal, and held in
  memory only while the app runs.
- When you add, edit, complete, or delete a task, the change is sent directly
  from your computer to the Google Tasks API.
- There are no analytics, no telemetry, no tracking, no advertising, and no
  crash reporting in guppy.

## Data storage

- guppy stores **no** task data on disk.
- A Google OAuth token (used to keep you signed in) is stored on your own
  computer at `~/.config/guppy/token.json` with owner-only file permissions
  (`0600`). It never leaves your device except to authenticate requests to
  Google.
- guppy's developers operate no servers and receive no copies of your data.

## Data sharing

guppy does not share, sell, rent, or transfer your data to anyone. The only
network communication is between your computer and Google's API servers.

## Google API Limited Use disclosure

guppy's use and transfer to any other app of information received from Google
APIs adheres to the
[Google API Services User Data Policy](https://developers.google.com/terms/api-services-user-data-policy),
including the Limited Use requirements.

## Deleting your data

There is nothing to delete on our side — we hold none of your data. To stop
guppy from accessing your Google Tasks:

1. Revoke guppy's access at
   [myaccount.google.com/permissions](https://myaccount.google.com/permissions), and
2. Delete the local token: `rm -rf ~/.config/guppy`

## Children's privacy

guppy requires a Google account and is not directed at children under 13.

## Changes to this policy

If this policy changes, the updated version will be published in the project
repository with a new effective date.

## Contact

Questions about this policy: `kartikssj@gmail.com` or open an issue at
`https://github.com/kartikssj/guppy`.
