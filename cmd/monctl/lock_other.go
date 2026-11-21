//go:build !unix

package main

func lockMonitor() func() { return func() {} }
