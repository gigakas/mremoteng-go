// Package credential provides Go ports of mRemoteNG's ExternalConnectors:
// clients for the external secret stores and address providers a
// connection can be configured to resolve its username/password or
// hostname from at connect time (connection.ExternalCredentialProvider,
// connection.ExternalAddressProvider), rather than storing them locally
// in the encrypted connections file.
//
// Each provider lives in its own subpackage (vault, delinea,
// passwordstate, onepassword, awsec2) since their APIs, auth schemes and
// request/response shapes have nothing in common beyond "fetch a
// secret". This package holds only the one shared result type.
//
// No UI: internal/ui/properties.go already renders
// ExternalCredentialProvider/ExternalAddressProvider as plain enum
// fields (stage 3.4); a picker that actually calls these clients from
// the UI is future work, not this stage's scope (see the blueprint's
// stage 3.7 parallelism note).
//
// internal/connection's model has fields for the Vault/OpenBao provider
// specifically (VaultOpenbaoMount/Role/SecretEngine) and for AWS address
// resolution (EC2InstanceID/EC2Region), but no per-provider secret
// identifier field for Delinea Secret Server, Passwordstate or
// 1Password — the original C# app's exact field-reuse convention for
// those three (which existing field holds the secret ID to look up) is
// not recoverable from this repository. Rather than guess and risk
// silently misreading the wrong field, every client in this package
// takes its secret identifier as an explicit parameter; wiring a
// connection.ConnectionInfo's fields to that parameter is deferred to
// whoever integrates this package with the UI/protocol layer, with the
// authoritative C# source in hand to confirm the convention.
package credential

// Credential is a resolved username/password pair, the common return
// shape every provider in this package produces.
type Credential struct {
	Username string
	Password string
}
