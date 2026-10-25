# Fictional forwarding sources

These are synthetic configuration texts, not observations from a machine.
`iptables-save.txt` covers custom chains, a loopback DNAT destination, OUTPUT
REDIRECT, a non-loopback destination and mangle TPROXY. `nft.json` covers native
IPv4/IPv6 loopback DNAT, REDIRECT, counters and an indirect map destination.
The reader retains candidates and full context without claiming that they are
reachable or effective paths. See the [collector contract](../../../../docs/specifications/prototype/forwarding-collector.md).

Tests read these fixtures only. All generated report/browser artifacts are written
to temporary directories. No firewall configuration is changed or loaded.
