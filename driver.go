// The Suricata driver: the entire product-specific surface of the
// integration, answering the six-verb contract from opamp-agent/driver.
//
// A vendor reading this file should finish understanding that
// identity / render / validate / apply / verify / health / effective-config
// is ALL their product needs to expose to be OpAMP-managed.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/agardnerIT/opamp-agent/driver"
)

type SuricataDriver struct {
	SocketPath string // suricatasc unix socket
	RulesPath  string // the rules file the operator's suricata.yaml references
}

// identity: who are you, and which files do you own?
func (d SuricataDriver) Identity() driver.Identity {
	v, _ := socketCall(d.SocketPath, "version")
	var ver string
	_ = json.Unmarshal(v, &ver)
	return driver.Identity{
		ServiceName: "suricata",
		Version:     ver,
		OwnedFiles:  []string{d.RulesPath},
		NonIdentifying: map[string]string{
			"suricata.version": ver,
		},
	}
}

// render: the remote config body IS the rules file content (PoC convention).
func (d SuricataDriver) Render(cfg []byte) (driver.Rendered, error) {
	if len(cfg) == 0 {
		return driver.Rendered{}, fmt.Errorf("empty config body")
	}
	return driver.Rendered{Artifacts: []driver.Artifact{
		{Path: d.RulesPath, Body: cfg},
	}}, nil
}

// validate: honest refusal — Suricata validates on reload; that is what the
// verify seam and LKG recovery are for.
func (d SuricataDriver) Validate(r driver.Rendered) error { return driver.ErrUnsupported }

// apply: activate the already-written artifacts (the bridge writes files; the
// driver only triggers the product's own apply mechanism). NOTE: the reload
// call's return value lies ("done" even for broken rules) — Verify is the
// only source of truth.
func (d SuricataDriver) Apply(ctx context.Context, r driver.Rendered) error {
	_, err := socketCall(d.SocketPath, "ruleset-reload-rules")
	return err
}

// verify: ruleset-stats + ruleset-failed-rules → line-precise rejection
// reasons. This is the demo moment: per-agent rejection feedback.
func (d SuricataDriver) Verify(ctx context.Context, r driver.Rendered) (bool, string) {
	msg, err := socketCall(d.SocketPath, "ruleset-stats")
	if err != nil {
		return false, "verify: " + err.Error()
	}
	var stats []rulesetStat
	if err := json.Unmarshal(msg, &stats); err != nil || len(stats) == 0 {
		return false, fmt.Sprintf("verify: ruleset-stats parse: %v (%s)", err, string(msg))
	}
	if stats[0].RulesFailed == 0 {
		return true, ""
	}
	reason := fmt.Sprintf("%d rule(s) rejected:", stats[0].RulesFailed)
	if msg, err := socketCall(d.SocketPath, "ruleset-failed-rules"); err == nil {
		var fr []failedRule
		_ = json.Unmarshal(msg, &fr)
		for _, r := range fr {
			reason += fmt.Sprintf("\n%s:%d %s", r.Filename, r.Line, r.Rule)
		}
	}
	return false, reason
}

// health: alive = socket reachable and uptime readable.
func (d SuricataDriver) Health(ctx context.Context) driver.Health {
	msg, err := socketCall(d.SocketPath, "uptime")
	if err != nil {
		return driver.Health{Healthy: false, Detail: "suricatasc unreachable: " + err.Error()}
	}
	var up int64
	if err := json.Unmarshal(msg, &up); err != nil {
		return driver.Health{Healthy: false, Detail: "uptime parse: " + err.Error()}
	}
	return driver.Health{
		Healthy:   up > 0,
		StartTime: time.Now().Add(-time.Duration(up) * time.Second),
		Detail:    fmt.Sprintf("uptime %ds", up),
	}
}

// effective config: hash of the owned file — stated, not claimed.
func (d SuricataDriver) EffectiveConfig(ctx context.Context) map[string][]byte {
	b, err := os.ReadFile(d.RulesPath)
	if err != nil {
		return nil
	}
	sum := sha256.Sum256(b)
	return map[string][]byte{"suricata.rules.sha256": sum[:]}
}
