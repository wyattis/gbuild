# gbuild
This is a tool to make cross-compiling multiple release binaries simple

## Install
```
go install github.com/wyattis/gbuild
```

## Basic usage
By default, all first-class platforms are built. From the project root run:
```bash
gbuild build                  # alias for `gbuild build first-class`
gbuild build first-class web  # build all first class and web platforms (js/wasm)
gbuild build all              # build all supported platforms
gbuild build cgo -mobile      # only build platforms w/ cgo support except mobile
gbuild build second-class     # only build second class platforms
```

### Combining aliases
Aliases are evaluated using set [union] and [complement] operations in the order they appear.

Examples:
`gbuild build android arm` is the union of all android and arm targets which would result in 10 binaries.
`gbuild build cgo -apple` is the set of cgo targets excluding all apple targets 
and would create 35 binaries.
`gbuild build second-class windows -ios -web` would build all of the second-class and windows targets while excluding ios and the web

See all of the available aliases with `gbuild list`

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
gbuild build -clean
```

### Use a custom release directory
```bash
gbuild build -o dist
```

### Passing additional args to build command
Separate the gbuild arguments from the "go build" arguments using "--"
```
gbuild build -- --ldflags '-extldflags "-Wl,--allow-multiple-definition"'
```

[union]: https://en.wikipedia.org/wiki/Union_(set_theory)
[difference]: https://en.wikipedia.org/wiki/Difference_(set_theory)