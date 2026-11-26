# opamp-suricata — Suricata under standard OpAMP

[![opamp-agent](https://img.shields.io/badge/built%20on-opamp--agent-blue)](https://github.com/agardnerIT/opamp-agent)

![Warning: Entirely vibecoded](https://img.shields.io/badge/Warning-Entirely%20vibecoded-orange?style=for-the-badge)

A complete, working **vendor-artifact reference**: the generic OpAMP bridge
([opamp-agent](https://github.com/agardnerIT/opamp-agent)) plus the one
product-specific driver that teaches Suricata to be OpAMP-managed. Watch an
IDS get its detection rules pushed by any OpAMP server — and get per-agent
rejection feedback when a rule is bad.

> This repo is primarily a **reference implementation**. If you maintain
> another product, the fastest way to understand the integration cost is to
> read `driver.go`: it is the entire product-specific surface. If you'd
> rather implement OpAMP natively inside your own agent, see
> [opamp-agent's vendor note](https://github.com/agardnerIT/opamp-agent#are-you-a-vendor-trying-to-add-opamp-support-for-your-product) —
> native is the better end state.
>
> Second reference implementation:
> [**opamp-falco**](https://github.com/agardnerIT/opamp-falco) — the same
> three-file shape for a product whose apply is an API call (`POST /reload`)
> and whose verify is a server-side generation counter, rather than
> Suricata's stats polling. Read them side by side to see which parts of a
> driver are genuinely product-specific.

## What's here (three files)

| File | Role |
|---|---|
| `driver.go` | The six verbs, answered for Suricata: identity, render, validate (honest refusal — Suricata validates on reload), apply (`ruleset-reload-rules` over the control socket), verify, health, effective config |
| `socket.go` | The `suricatasc` unix-socket protocol, captured from the real binary: every connection opens with `{"version":"0.2"}\n`, then one command per line; each response is a single line of JSON |
| `main.go` | Flags + `bridge.Run(...)` — the whole composition |

Everything else — OpAMP transport, enrollment, health reporting, and
**last-known-good recovery** (a rejected push never leaves the IDS with zero
rules loaded) — is [opamp-agent](https://github.com/agardnerIT/opamp-agent)'s
job.

## What a config push looks like

- Good rules → `RemoteConfigStatuses_APPLIED`.
- A broken rule → `RemoteConfigStatuses_FAILED` with Suricata's exact
  rejection (rule text + file + line number) while the previous ruleset
  stays live.

## Run it

```
suricata -c suricata.yaml --unix-socket=/tmp/suricata.sock --set pcap.0.interface=<iface>
go build . && ./opamp-suricata -token opamp_agent_... -server ws://<opamp-server>/v1/opamp
```

Requires Suricata ≥ 8 (unix-socket control), and an OpAMP server (any
spec-compliant one works; this was proven against
[opamp-server-py](https://github.com/agardnerIT/opamp-server-py)).

## Hard-won facts baked into the code

- The reload call's return value **lies** (`"done"` even for broken rules) —
  verification polls `ruleset-stats` + `ruleset-failed-rules`, never trusts
  the apply response.
- A rejected full-file push **disarms the IDS** (`rules_loaded: 0`) unless
  the bridge restores last-known-good before re-applying.
- The socket protocol needs the version handshake or every command gets
  instant EOF.

## License

Apache-2.0 (same as opamp-agent).
