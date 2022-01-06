# gbuild
Simple tool to cross-compile release binaries

## Install
```
go install github.com/wyattis/gbuild
```

## Basic usage
By default, all first-class platforms are built. From the project root run:
```bash
gbuild build                  # alias for `gbuild first-class`
gbuild build first-class web  # build all first class and web platforms (js/wasm)
gbuild build all              # build all supported platforms
gbuild build cgo              # only build platforms w/ cgo support
gbuild build second-class     # only build second class platforms
```


## Options
```
-bundle-template string
      template to use for each bundle (default "{{.NAME}}_{{.GOOS}}_{{.GOARCH}}{{.ZIP}}")
-clean
      clean the output directory before building
-name string
      executable name
-name-template string
      template to use for each file (default "{{.NAME}}{{.EXT}}")
-o string
      output directory (default "release")
```


## Other examples

### Clean release directory before building
```bash
gbuild -clean
```

### Use a custom release directory
```bash
gbuild -o dist
```

### Passing additional args to build command
Separate the gbuild arguments from the "go build" arguments using "--"
```
gbuild -- --ldflags '-extldflags "-Wl,--allow-multiple-definition"'
```

## TODO
- [ ] alias joining rules (cgo -second-class)
- [ ] 