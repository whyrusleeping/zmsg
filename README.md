# zmsg
A zero knowledge messaging system built on zcash.

zmsg uses the encrypted memo field of zcash sheilded transactions to send
messages to other parties. The sent messages get stored in the zcash blockchain
and the recipient can check for messages at any time (no need to be online at
the same time). Since the messages get stored in the blockchain, they are on
*every* full zcash node. The implication here is that its not possible to tell
who the target of any given message is.

Currently, each message sends 0.0001 ZEC. You can change this value by setting
the `--txval` flag on `sendmsg`.

## Installation
First, make sure you have [go](https://golang.org/doc/install) installed, then:
```sh
go get github.com/whyrusleeping/zmsg
```

## Usage
Note: To use zmsg, you'll need a running zcash daemon, a z_address, and some
spare ZEC in that address.

### sendmsg
To send a message, use `zmsg sendmsg`:
```sh
$ export TARGET_ZADDR=zchfvC6iubfsAxaNrbM4kkGDSpwjafECjqQ1BZBFXtotXyXARz2NoYRVEyfLEKGCFRY7Xfj2Q3jFueoHHmQKb63C3zumYnU
$ zmsg sendmsg -to=$TARGET_ZADDR "Hello zmsg! welcome to pretty secure messaging"
message: "Hello zmsg! welcome to pretty secure messaging"
sending message from <your z_addr>
sending message...
Message sent successfuly!
message sent! (txid = <transaction id>)
```

Note that this will take about a minute to compute the zero-knowledge proof,
and another few minutes before the transaction gets propagated and confirmed
for the other side to see it.

### check
To check for messages, run `zmsg check`:

```sh
================================================================================
> Got 2 messages.
================================================================================
| Message #0 (val = 0.000010)
| To: zchfvC6iubfsAxaNrbM4kkGDSpwjafECjqQ1BZBFXtotXyXARz2NoYRVEyfLEKGCFRY7Xfj2Q3jFueoHHmQKb63C3zumYnU
| Date: 2016-11-11 17:36:31 -0800 PST
|
|  This is a test of zmsg, hello everyone!
================================================================================
| Message #1 (val = 0.000010)
| To: zchfvC6iubfsAxaNrbM4kkGDSpwjafECjqQ1BZBFXtotXyXARz2NoYRVEyfLEKGCFRY7Xfj2Q3jFueoHHmQKb63C3zumYnU
| Date: 2016-11-11 17:44:44 -0800 PST
|
|  This is message number 'two', i'm sitting in a coffee shop. Don't tell anyone.
================================================================================
```

## Send me a message!
If you're trying this out and want to say hi, send me a message at `zchfvC6iubfsAxaNrbM4kkGDSpwjafECjqQ1BZBFXtotXyXARz2NoYRVEyfLEKGCFRY7Xfj2Q3jFueoHHmQKb63C3zumYnU`.

## License
MIT, whyrusleeping
