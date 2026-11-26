// suricatasc unix-socket RPC. This file is 100% product knowledge: the wire
// protocol was captured from the real binary (see poc-suricata/captures/03):
// every connection opens with {"version":"0.2"}\n, then one
// {"command":"..."}\n per call, each response a single line of JSON.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

func socketCall(path, command string) (json.RawMessage, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(60 * time.Second))
	r := bufio.NewReader(conn)
	if _, err := conn.Write([]byte(`{"version":"0.2"}` + "\n")); err != nil {
		return nil, err
	}
	if _, err := r.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("suricatasc handshake: %w", err)
	}
	req, _ := json.Marshal(struct {
		Command string `json:"command"`
	}{command})
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return nil, err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	var resp struct {
		Return  string          `json:"return"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, fmt.Errorf("suricatasc response %q: %w", line, err)
	}
	if resp.Return != "OK" {
		return nil, fmt.Errorf("suricatasc %s: %s", command, line)
	}
	return resp.Message, nil
}

// rulesetStats mirrors: [{"id":0,"rules_loaded":N,"rules_failed":N,"rules_skipped":N}]
type rulesetStat struct {
	ID           int64 `json:"id"`
	RulesLoaded  int64 `json:"rules_loaded"`
	RulesFailed  int64 `json:"rules_failed"`
	RulesSkipped int64 `json:"rules_skipped"`
}

// failedRule mirrors: {"tenant_id","rule","filename","line"}
type failedRule struct {
	Rule     string `json:"rule"`
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}
