# party-line

> A demo app for Protocol Labs Hack Week 2021. Don't let NATs stop your chats!


## Wat

A chat app for sending text and audio messages to connected peers via libp2p. 

Work in Progress!

## Install / Usage

You'll need the Opus audio codec:

- macos:
  - `brew install opus`
- linux (debian style):
  - `apt install libopus-dev`
    
On linux you'll also need `libasound2-dev`:

- `apt install libasound2-dev`

Then run `make` to build the server and frontend. This will create a `party-line` binary:

```
make
./party-line
```

This should open a window with the UI. If you prefer, you can run the app server "headless" and open a browser
yourself (handy for DevTools).

```
./party-line -headless
```

