# poker-chan-api 
プランニングポーカーちゃんのバックエンド処理
| # | |
| ---- | ---- |
| Language | Go 1.23 |
| Router | gorilla/websocket |
| etc | google/uuid |

# 使用例
wscatを使用する場合

## インストール
```
npm install -g wscat
```

## チャットサーバ立ち上げ
```
go run main.go
```

## 接続の確立 (複数のターミナルから)
```
npx wscat -c ws://localhost:8080/ws?id=1
```

## 名前入力
```
{"name":"alice"}
```

## 投票
```
{"vote":"5"}
```

## プランニング対象変更
```
{"title":"project abc"}
```

## 投票リセット
```
{"reset":true}
```

## 開示
```
{"reveal":true}
```

疎通できればＯＫ
