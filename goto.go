package main

import (
	"github.com/RecursiveForest/goty" //fork and maintain building copy
	"fmt"
	"os"
)


func main() {
	con, err := goty.Dial("irc.freenode.net:6666", "ihearduliekbawts", "unoudo")
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %s\n", err.Error())
	}
	con.Write <- "JOIN #reddit-anime"
	con.Write <- "PRIVMSG #reddit-anime :aww yiss?\r\n"
	for {
		res, ok := <-con.Read
		if !ok { break }
		fmt.Fprintf(os.Stdout, "%s\n", res)
	}
	con.Close()
}

