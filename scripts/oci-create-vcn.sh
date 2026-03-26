#!/usr/bin/env bash
# Create a simple public VCN + subnet in OCI (CLI only).
# Prerequisites: oci configured, COMPARTMENT_ID and OCI_CLI_REGION (or ~/.oci/config region) set.
#
# Usage:
#   export COMPARTMENT_ID="ocid1.tenancy.oc1..xxx"
#   export OCI_CLI_REGION=us-ashburn-1   # optional if set in config
#   ./scripts/oci-create-vcn.sh
#
# Optional:
#   export VCN_CIDR=10.0.0.0/16
#   export SUBNET_CIDR=10.0.0.0/24
# Subnet: uses first AD in this region (CLI versions without --subnet-domain-type REGIONAL).
# Or set explicitly: SUBNET_USE_AD=1 and AD="gpdV:US-PHOENIX-1-AD-1" (must match OCI_CLI_REGION).

set -euo pipefail

: "${COMPARTMENT_ID:?Set COMPARTMENT_ID (tenancy or compartment OCID)}"

VCN_CIDR="${VCN_CIDR:-10.0.0.0/16}"
SUBNET_CIDR="${SUBNET_CIDR:-10.0.0.0/24}"
VCN_NAME="${VCN_NAME:-gogents-vcn}"
SUBNET_NAME="${SUBNET_NAME:-gogents-public}"

# DNS labels: lowercase alphanumeric, max 15 chars, unique in tenancy for VCN
VCN_DNS="${VCN_DNS:-gogentsvcn}"
SUBNET_DNS="${SUBNET_DNS:-gogentspub}"

echo "==> Creating VCN $VCN_NAME ($VCN_CIDR)"
VCN_ID=$(oci network vcn create \
  --compartment-id "$COMPARTMENT_ID" \
  --cidr-blocks "[\"$VCN_CIDR\"]" \
  --display-name "$VCN_NAME" \
  --dns-label "$VCN_DNS" \
  --wait-for-state AVAILABLE \
  --query 'data.id' --raw-output)
echo "    VCN_ID=$VCN_ID"

echo "==> Creating Internet Gateway"
IGW_ID=$(oci network internet-gateway create \
  --compartment-id "$COMPARTMENT_ID" \
  --vcn-id "$VCN_ID" \
  --display-name "${VCN_NAME}-igw" \
  --is-enabled true \
  --wait-for-state AVAILABLE \
  --query 'data.id' --raw-output)
echo "    IGW_ID=$IGW_ID"

echo "==> Updating default route table (0.0.0.0/0 -> IGW)"
RT_ID=$(oci network vcn get --vcn-id "$VCN_ID" --query 'data."default-route-table-id"' --raw-output)
oci network route-table update \
  --rt-id "$RT_ID" \
  --route-rules "[{\"destination\":\"0.0.0.0/0\",\"destinationType\":\"CIDR_BLOCK\",\"networkEntityId\":\"$IGW_ID\"}]" \
  --force
echo "    RT_ID=$RT_ID"

echo "==> Updating default security list (SSH 22, HTTP 80, HTTPS 443)"
SL_ID=$(oci network vcn get --vcn-id "$VCN_ID" --query 'data."default-security-list-id"' --raw-output)
EGRESS_FILE=$(mktemp)
INGRESS_FILE=$(mktemp)
cleanup() { rm -f "$EGRESS_FILE" "$INGRESS_FILE"; }
trap cleanup EXIT
oci network security-list get --security-list-id "$SL_ID" --query 'data."egress-security-rules"' > "$EGRESS_FILE"
cat > "$INGRESS_FILE" <<'INGRESS_EOF'
[
  {"source": "0.0.0.0/0", "sourceType": "CIDR_BLOCK", "protocol": "6", "tcpOptions": {"destinationPortRange": {"min": 22, "max": 22}}, "description": "SSH"},
  {"source": "0.0.0.0/0", "sourceType": "CIDR_BLOCK", "protocol": "6", "tcpOptions": {"destinationPortRange": {"min": 80, "max": 80}}, "description": "HTTP"},
  {"source": "0.0.0.0/0", "sourceType": "CIDR_BLOCK", "protocol": "6", "tcpOptions": {"destinationPortRange": {"min": 443, "max": 443}}, "description": "HTTPS"}
]
INGRESS_EOF
oci network security-list update \
  --security-list-id "$SL_ID" \
  --ingress-security-rules "file://$INGRESS_FILE" \
  --egress-security-rules "file://$EGRESS_FILE" \
  --force
echo "    SL_ID=$SL_ID"

echo "==> Creating public subnet $SUBNET_NAME ($SUBNET_CIDR)"
SUBNET_ARGS=(
  --compartment-id "$COMPARTMENT_ID"
  --vcn-id "$VCN_ID"
  --cidr-block "$SUBNET_CIDR"
  --display-name "$SUBNET_NAME"
  --dns-label "$SUBNET_DNS"
  --prohibit-public-ip-on-vnic false
  --wait-for-state AVAILABLE
)
if [[ "${SUBNET_USE_AD:-}" == "1" && -n "${AD:-}" ]]; then
  SUBNET_ARGS+=(--availability-domain "$AD")
else
  # Older OCI CLI has no --subnet-domain-type; use first AD in current region
  SUBNET_AD=$(oci iam availability-domain list --compartment-id "$COMPARTMENT_ID" --query 'data[0].name' --raw-output)
  echo "    (subnet AD: $SUBNET_AD)"
  SUBNET_ARGS+=(--availability-domain "$SUBNET_AD")
fi

SUBNET_ID=$(oci network subnet create "${SUBNET_ARGS[@]}" --query 'data.id' --raw-output)
echo "    SUBNET_ID=$SUBNET_ID"

echo
echo "Done. Export for instance launch:"
echo "export VCN_ID=$VCN_ID"
echo "export SUBNET_ID=$SUBNET_ID"
