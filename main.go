// Demo composition: the vendor's binary = the generic bridge + their driver.
// This is the whole "Rung B" artifact — the only vendor-written glue is the
// flags below. Everything else is github.com/agardnerIT/opamp-agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/agardnerIT/opamp-agent/bridge"
)

func main() {
	server := flag.String("server", "ws://127.0.0.1:4320/v1/opamp", "OpAMP endpoint")
	token := flag.String("token", os.Getenv("OPAMP_TOKEN"), "agent bearer token")
	rules := flag.String("rules", "/opt/homebrew/var/lib/suricata/rules/suricata.rules", "owned rules file")
	socket := flag.String("socket", "/tmp/suricata-poc.sock", "suricatasc unix socket")
	flag.Parse()

	drv := SuricataDriver{SocketPath: *socket, RulesPath: *rules}
	if err := bridge.Run(context.Background(), bridge.Options{
		ServerURL:       *server,
		Token:           *token,
		InstanceUIDFile: "instance.uid",
		Logger:          stdoutLogger{},
	}, drv); err != nil {
		fmt.Fprintln(os.Stderr, "opamp-suricata:", err)
		os.Exit(1)
	}
}

type stdoutLogger struct{}

func (stdoutLogger) Debugf(format string, v ...any) {}
func (stdoutLogger) Errorf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, format, v...)
}
