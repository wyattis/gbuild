package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type ModConfig struct {
	Name      string
	GoVersion string
}

func parseMod(loc string) (mod ModConfig, err error) {
	f, err := os.Open(loc)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			return
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			_, name, found := StringCut(line, " ")
			if !found {
				err = errors.New("invalid module name in go.mod")
				return
			}
			mod.Name = name
			return
		}
	}
	return
}

type Distribution struct {
	GOOS         string
	GOARCH       string
	FirstClass   bool
	CgoSupported bool
}

func (d Distribution) String() string {
	return d.GOOS + "/" + d.GOARCH
}

type DistributionSet []Distribution

func (d *DistributionSet) Set(val string) error {
	vals := StringSplitMany(val, ",", " ")
	for _, v := range vals {
		os, arch, _ := StringCutAny(v, "/", "\\")
		*d = append(*d, Distribution{GOOS: os, GOARCH: arch})
	}
	return nil
}

func (d DistributionSet) String() (res string) {
	for i, v := range d {
		if i != 0 {
			res += ", "
		}
		res += v.String()
	}
	return
}

func (d DistributionSet) Only(systems ...string) (res DistributionSet) {
	for _, dist := range d {
		if StringSliceContains(systems, dist.GOOS) {
			res = append(res, dist)
		}
	}
	return
}

func (d DistributionSet) WithoutArch(archs ...string) (res DistributionSet) {
	for _, dist := range d {
		if !StringSliceContains(archs, dist.GOARCH) {
			res = append(res, dist)
		}
	}
	return
}

func (d DistributionSet) OnlyArch(archs ...string) (res DistributionSet) {
	for _, dist := range d {
		if StringSliceContains(archs, dist.GOARCH) {
			res = append(res, dist)
		}
	}
	return
}

func (d DistributionSet) Copy() (res DistributionSet) {
	res = make(DistributionSet, len(d))
	copy(res, d)
	return
}

func (d DistributionSet) Has(val Distribution) bool {
	for _, dist := range d {
		if dist == val {
			return true
		}
	}
	return false
}

func (d DistributionSet) Union(other DistributionSet) (res DistributionSet) {
	res = d.Copy()
	for _, oDist := range other {
		if !res.Has(oDist) {
			res = append(res, oDist)
		}
	}
	return
}

func (d DistributionSet) Difference(other DistributionSet) (res DistributionSet) {
	res = make(DistributionSet, 0)
	for _, dist := range d {
		if !other.Has(dist) {
			res = append(res, dist)
		}
	}
	return
}

func getAllDistributions() (res DistributionSet, err error) {
	buf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(context.Background(), "go", "tool", "dist", "list", "-json")
	cmd.Stdout = buf
	cmd.Stderr = errBuf
	if err = cmd.Run(); err != nil {
		return
	}
	err = json.Unmarshal(buf.Bytes(), &res)
	return
}
