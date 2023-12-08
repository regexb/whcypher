## Developing

### Source wasm_exec.js

```sh
$ cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" wasm/static
```

### Compile

```sh
$ GOOS=js GOARCH=wasm go build -o wasm/static/main.wasm wasm/main.go
```

