```
go mod tidy
go run . -addr 127.0.0.1:8000
```

make sure this code correct:

`u := url.URL{Scheme: "ws", Host: *addr, Path: "/your/ws/handler"}`