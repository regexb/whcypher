[![Netlify Status](https://api.netlify.com/api/v1/badges/8acdd1f6-9dcb-4681-81c3-b6552d5426a9/deploy-status)](https://app.netlify.com/sites/whcypher/deploys)

## Developing

### Source wasm_exec.js

```sh
$ cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" wasm/static
```

### Compile

```sh
$ GOOS=js GOARCH=wasm go build -o wasm/static/main.wasm wasm/main.go
```

