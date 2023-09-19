package main

import "fmt"

const (
	VersionMajor = 1
	VersionMinor = 0
	VersionPatch = 0
)

func VersionString() string {
	return fmt.Sprint("v", VersionMajor, ".", VersionMinor, ".", VersionPatch)
}
