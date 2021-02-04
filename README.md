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

When you launch the app, you should see some output like this:

```
setting up libp2p host...
server peer id is:  QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
-----------------------------------------------------------------------------------------------------------------------------------
server addrs are:
/ip4/192.168.64.1/tcp/57328/p2p/QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
/ip4/127.0.0.1/tcp/57328/p2p/QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
/ip4/54.255.209.104/tcp/12001/p2p/Qma71QQyJN7Sw7gz1cgJ4C66ubHmvKqBasSegKRugM5qo6/p2p-circuit/p2p/QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
/ip4/54.255.209.104/udp/12001/quic/p2p/Qma71QQyJN7Sw7gz1cgJ4C66ubHmvKqBasSegKRugM5qo6/p2p-circuit/p2p/QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
-----------------------------------------------------------------------------------------------------------------------------------

 your NAT device supports NAT traversal via hole punching for TCP connections
------------------------------------------------------------------------------------------------------------------------------------
accepting connections now
starting UI server on localhost:7777
```

## Connecting to peers

I haven't added a UI to connect to peers yet, so you have to know who you want to chat with when you launch the app.
Just pass in the peer id of the peer you want to connect to on the cli after any flags:

```
./party-line QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
```

If you get an error about not being able to find addresses, you can pass a full multiaddr instead, e.g.:

```
./party-line /ip4/192.168.64.1/tcp/57328/p2p/QmP1qS4TvreM33hkgubH1RCYQYqm3PaLDV6PENYmTd39PG
```
