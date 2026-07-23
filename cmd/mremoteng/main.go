package main

import (
	"fmt"

	// Blank-imported so each protocol backend's init() registers itself
	// with internal/protocol's factory (see internal/protocol/factory.go's
	// Register doc comment). Add a line here for every protocol this
	// binary should support; a build that wants a smaller footprint can
	// drop backends by removing their import.
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/raw"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rlogin"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/serial"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
)

func main() {
	fmt.Println("mremoteng-go: skeleton, not yet functional")
}
