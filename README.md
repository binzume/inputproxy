# WebSocket経由でマウスやキーボード入力をするやつ

https://github.com/binzume/webrtc-rdp 用につくったもの．

起動するとWebSocketのURLが出力されるので，デスクトップのキャスト画面の入力ボックスに貼り付けてください．


## Usage

```
go build
./inputproxy -port 9000
```

デフォルトでは localhost:9000 で待ち受けます．URLは毎回変わるので固定したい場合は secret オプションを指定してください．

## License

MIT
