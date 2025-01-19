package main

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/term"
)

var _interrupted bool

func main() {
	handleSignals()
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	var i int
	for range ticker.C {
		if _interrupted {
			continue
		}
		i++
		secs := int(time.Since(start).Seconds())
		w, h, _ := term.GetSize(0)
		s := fmt.Sprintf("\x1b[32m[%04d]\x1b[0m %dx%d", secs%10000, w, h)
		if i%8 >= 4 {
			// Print a progress bar.
			if w == 0 {
				w = 80
			}
			s += " "
			w -= len(s) - len("\x1b[32m\x1b[0m")
			if w < 18 {
				w = 18
			}
			filled := (w - 2) * (i%8 - 3) / 4
			s += "[" + strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", w-2-filled) + "]"
			if i%8 == 7 {
				s += "\n"
			} else {
				s += "\r"
			}
		} else if i%8 == 1 {
			s += " test program with term size, progress bar and color\n"
		} else {
			s += "\n"
		}
		fmt.Print(s)
	}
}
