package main

import (
	"fmt"

	"github.com/tools/godep/Godeps/_workspace/src/github.com/kr/pretty"
)

func debugln(a ...interface{}) (int, error) {
	if debug {
		return fmt.Println(a...)
	}
	return 0, nil
}

func debugf(format string, a ...interface{}) (int, error) {
	if debug {
		return fmt.Printf(format, a...)
	}
	return 0, nil
}

func pp(a ...interface{}) (int, error) {
	if debug {
		return pretty.Print(a...)
	}
	return 0, nil
}

func ppln(a ...interface{}) (int, error) {
	if debug {
		return pretty.Println(a...)
	}
	return 0, nil
}

func ppf(format string, a ...interface{}) (int, error) {
	if debug {
		return pretty.Printf(format, a...)
	}
	return 0, nil
}
