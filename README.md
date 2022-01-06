# gbuild
Simple tool to cross-compile release binaries

## Install
```
go install github.com/wyattis/gbuild
```

## Basic usage
From the project root:
```
gbuild
```

## Other examples

### Passing additional args to build command
Separate the gbuild arguments from the "go build" arguments using "--"
```
gbuild -- --ldflags '-extldflags "-Wl,--allow-multiple-definition"'
```

## TODO
- [x] parse package name from module file
- [x] bundle as zip
- [ ] custom aliases
- [ ] custom filters
- [ ] use output from `go tool dist list -json` for targets