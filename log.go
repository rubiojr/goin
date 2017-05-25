package main

import "fmt"

func Debugf(msg string, args ...interface{}) {
	if *isDebug {
		fmt.Printf(msg+"\n", args...)
	}
}
