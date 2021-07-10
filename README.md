# screm

Simple background service which answers a sound based on a Twitch chat command. Support whitelist Inspired
from https://github.com/bfroggio/screm without keyboard shortcuts, sound category and write messages.

## Usage

1. Create a `config.toml` file as outlined in the ["Config File" section](#config-file) below
1. Put any sound effects you want to use in `./sounds`. The file name will be the command the to trigger that sound
1. Launch `screm.exe` by double-clicking on it
1. Try this with `!ding` command in your Twitch chat

## Config File

`config.toml` should be in the same directory as the `screm.exe` binary. There are a few properties that can go in
your `config.toml` file. Properties listed below are optional unless otherwise noted. See `sample-config.toml` for an
example with fake configuration values.

- `twitch_username` (required): The username for your Twitch account/channel. Screm can't read your channel's chat
  messages without this.
- `twitch_authorized_users`: A space-separated list of Twitch usernames for users authorized to trigger sound effects on
  your stream from Twitch chat.
