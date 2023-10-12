package main

import (
	"flag"
	"fmt"

	"github.com/primevprotocol/mev-commit/examples/builderemulator/client"
)

func main() {
	// Define flags
	serverAddr := flag.String("serverAddr", "localhost:13524", "The server address in the format of host:port")

	flag.Parse()
	if *serverAddr == "" {
		fmt.Println("Please provide a valid server address with the -serverAddr flag")
		return
	}

	builderClient, err := client.NewBuilderClient(*serverAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = builderClient.ReceiveBids()
	if err != nil {
		fmt.Println(err)
		return
	}
}
