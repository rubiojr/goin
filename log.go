package main

import "fmt"

func Debugf(msg string, args ...interface{}) {
	if *isDebug {
		fmt.Printf(msg+"\n", args...)
	}
}

func printError(err string, args ...interface{}) error {
	if *isDebug {
		return fmt.Errorf(red("error: ")+err, args...)
	} else {
		fmt.Printf("X")
		return nil
	}
}
