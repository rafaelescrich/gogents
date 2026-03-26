#!/usr/bin/env bash
# Open inbound TCP 80 and 443 on an existing VCN default security list (and optionally an NSG).
# Use this if Let's Encrypt / gogents HTTPS fails with connection errors from the internet.
#
# Prerequisites: oci CLI configured.
#
# Usage (one of):
#   export VCN_ID=ocid1.vcn.oc1...
#   ./scripts/oci-open-web-ports.sh
#
# Or target the security list directly:
#   export SECURITY_LIST_ID=ocid1.securitylist.oc1...
#   ./scripts/oci-open-web-ports.sh
#
# If your instance uses a Network Security Group, also set:
#   export NSG_ID=ocid1.networksecuritygroup.oc1...
#
# Optional: add IPv6 ingress (::/0) as well
#   export INCLUDE_IPV6=1
#
set -euo pipefail

: "${OCI_CLI_SUPPRESS_FILE_PERMISSIONS_WARNING:=1}"

if [[ -z "${SECURITY_LIST_ID:-}" ]]; then
  : "${VCN_ID:?Set VCN_ID or SECURITY_LIST_ID}"
  SECURITY_LIST_ID="$(oci network vcn get --vcn-id "$VCN_ID" --query 'data."default-security-list-id"' --raw-output)"
  echo "==> Using default security list of VCN: $SECURITY_LIST_ID"
else
  echo "==> Security list: $SECURITY_LIST_ID"
fi

TMPDIR="$(mktemp -d)"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

oci network security-list get --security-list-id "$SECURITY_LIST_ID" \
  --query 'data."ingress-security-rules"' > "$TMPDIR/ingress.json"
oci network security-list get --security-list-id "$SECURITY_LIST_ID" \
  --query 'data."egress-security-rules"' > "$TMPDIR/egress.json"

python3 - "$TMPDIR/ingress.json" "${INCLUDE_IPV6:-0}" << 'PY'
import json, sys
path, inc_v6 = sys.argv[1], sys.argv[2] == "1"
with open(path) as f:
    ingress = json.load(f)

def has_tcp_from(rules, source, port):
    for r in rules:
        if r.get("source") != source:
            continue
        if str(r.get("protocol")) != "6":
            continue
        dr = (r.get("tcpOptions") or {}).get("destinationPortRange") or {}
        if dr.get("min") == port and dr.get("max") == port:
            return True
    return False

def add_rule(rules, source, port, desc):
    if not has_tcp_from(rules, source, port):
        rules.append({
            "source": source,
            "sourceType": "CIDR_BLOCK",
            "protocol": "6",
            "tcpOptions": {"destinationPortRange": {"min": port, "max": port}},
            "description": desc,
        })

add_rule(ingress, "0.0.0.0/0", 80, "HTTP (ACME / web)")
add_rule(ingress, "0.0.0.0/0", 443, "HTTPS")
if inc_v6:
    add_rule(ingress, "::/0", 80, "HTTP (ACME / web) IPv6")
    add_rule(ingress, "::/0", 443, "HTTPS IPv6")

with open(path, "w") as f:
    json.dump(ingress, f, indent=2)
PY

oci network security-list update \
  --security-list-id "$SECURITY_LIST_ID" \
  --ingress-security-rules "file://$TMPDIR/ingress.json" \
  --egress-security-rules "file://$TMPDIR/egress.json" \
  --force

echo "==> Updated security list (TCP 80, 443)."

if [[ -n "${NSG_ID:-}" ]]; then
  echo "==> Adding rules to NSG $NSG_ID"
  if [[ "${INCLUDE_IPV6:-0}" == "1" ]]; then
    cat > "$TMPDIR/nsg-rules.json" << 'NSG_EOF'
[
  {"description":"HTTP (ACME / web)","direction":"INGRESS","protocol":"6","source":"0.0.0.0/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":80,"max":80}}},
  {"description":"HTTPS","direction":"INGRESS","protocol":"6","source":"0.0.0.0/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":443,"max":443}}},
  {"description":"HTTP IPv6","direction":"INGRESS","protocol":"6","source":"::/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":80,"max":80}}},
  {"description":"HTTPS IPv6","direction":"INGRESS","protocol":"6","source":"::/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":443,"max":443}}}
]
NSG_EOF
  else
    cat > "$TMPDIR/nsg-rules.json" << 'NSG_EOF'
[
  {"description":"HTTP (ACME / web)","direction":"INGRESS","protocol":"6","source":"0.0.0.0/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":80,"max":80}}},
  {"description":"HTTPS","direction":"INGRESS","protocol":"6","source":"0.0.0.0/0","sourceType":"CIDR_BLOCK","tcpOptions":{"destinationPortRange":{"min":443,"max":443}}}
]
NSG_EOF
  fi
  oci network nsg rules add --nsg-id "$NSG_ID" --security-rules "file://$TMPDIR/nsg-rules.json"
  echo "==> NSG updated (skip if rules already existed — remove duplicates in console if needed)."
fi

echo "Done."
