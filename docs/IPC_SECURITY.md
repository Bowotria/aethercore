# AetherCore IPC Security Specification

**Document:** Layer 0 ↔ Layer 2 Inter-Process Communication Security  
**Version:** 1.0  
**Status:** Authoritative  

---

## 1. Overview

AetherCore separates the trusted Go kernel (Layer 0) from the untrusted execution environment (Layer 2 Rust Sandbox) via a strictly defined IPC boundary. This document specifies the security properties, protocol design, threat model, and enforcement mechanisms for all communication that crosses this boundary.

The security guarantee is:

> **No untrusted code executes inside the Layer 0 process space, ever.** All unknown tool invocations from the LLM are forwarded to the isolated Rust sidecar, which enforces capability restrictions before execution.

---

## 2. Transport: Unix Domain Socket (UDS)

### 2.1 Why Unix Domain Sockets?

| Property | TCP Loopback | Unix Domain Socket |
|---|---|---|
| Kernel permission check | No | **Yes — filesystem ACL** |
| Network stack traversal | Yes | **No** |
| Credential passing (`UCred`) | No | **Yes — PID/UID/GID** |
| Latency | ~50µs | **~5µs** |
| Port exhaustion risk | Yes | **No** |

The socket file is created at a deterministic, process-owned path:

```
$TMPDIR/aether-sandbox.sock
```

File permissions are set to `0600` (owner read/write only) on creation. The kernel enforces access control at the filesystem level — no process running as a different UID may connect.

### 2.2 Socket Lifecycle

1. The Rust sidecar starts first and binds the socket, removing any stale socket file from a previous (crashed) run.
2. The Go kernel connects **after** the sidecar is listening. If the socket is absent, the kernel retries with exponential backoff up to 5 seconds, then logs `sandbox_unavailable` and disables sandbox dispatch for the session.
3. On clean shutdown, the Rust sidecar removes the socket file before exiting.

---

## 3. Protocol: gRPC over Protocol Buffers

### 3.1 Schema

All IPC messages are defined in `proto/ipc.proto` and compiled to both Go and Rust bindings. The canonical schema:

```protobuf
service Sandbox {
  rpc ExecuteTool(ToolRequest) returns (ToolResponse);
}

message ToolRequest {
  string tool_name    = 1;  // Capability identifier from manifest.toml
  string payload_json = 2;  // Strictly validated JSON arguments
}

message ToolResponse {
  bool   success       = 1;
  string output_json   = 2;  // Structured result for the LLM context
  string error_message = 3;  // Human-readable error for kernel logging
}
```

Protocol Buffers enforce strict binary framing and schema versioning. Unknown fields are dropped, preventing injection via future field extensions.

### 3.2 Message Size Limits

The gRPC server enforces a maximum message size of **4 MB** to prevent memory exhaustion attacks via oversized payloads. This limit is configured on both the client (Go) and server (Rust) sides.

---

## 4. Authentication and Authorization

### 4.1 Peer Identity (Current: Phase 1)

In Phase 1, the transport is UDS with filesystem-ACL enforcement only. The Rust sidecar verifies the peer's UID via the `SO_PEERCRED` socket option on every new connection. Connections from UIDs other than the kernel's UID are rejected immediately.

```rust
// Rust: peer credential verification on accept
let cred = stream.peer_cred()?;
if cred.uid() != expected_kernel_uid {
    return Err(Status::permission_denied("invalid peer uid"));
}
```

### 4.2 Mutual TLS (Phase 2 — Day 15)

Phase 2 will replace the UDS transport security with full mTLS:
- Both kernel and sandbox present X.509 certificates signed by a locally-generated CA.
- Certificate private keys are generated at first boot and stored in a kernel-owned directory with `0400` permissions.
- All IPC connections are authenticated at the TLS handshake before any gRPC frames are exchanged.

---

## 5. Capability Enforcement

### 5.1 Manifest-Driven Authorization

Every tool that the sandbox may execute must be pre-declared in `manifest.toml`. The Rust sidecar cross-references `ToolRequest.tool_name` against this manifest before executing anything:

```toml
[[tools]]
name            = "fetch_url"
capabilities    = ["network"]
max_runtime_ms  = 5000
memory_limit_mb = 64
```

If `tool_name` is absent from the manifest, the sandbox returns:

```json
{"success": false, "error_message": "capability_denied: tool not declared in manifest"}
```

No code runs. No exceptions.

### 5.2 Capability Classes

| Capability | Grants Access To |
|---|---|
| `filesystem` | Read/write within declared path prefixes only |
| `network` | Outbound HTTP/HTTPS to declared hostnames only |
| `subprocess` | Spawning child processes (severely restricted) |
| `wasm` | Executing a WASM binary via the wasmtime engine |

Capabilities not listed in the tool's manifest entry are denied at the kernel level before any IPC message is sent.

---

## 6. Resource Limits

### 6.1 cgroup v2 Memory Enforcement (Linux)

When the sandbox boots, it applies individual cgroup v2 memory limits per tool (sourced from `manifest.toml`). The cgroup is created under `/sys/fs/cgroup/aether/<tool_name>/` with:

- `memory.max` = `memory_limit_mb * 1024 * 1024` (hard limit — OOM killer triggers)
- `memory.swap.max` = `0` (swap is disabled — prevents memory limit bypass)

The current sandbox process is added to the cgroup, which is automatically inherited by any child processes or threads.

### 6.2 Execution Timeout

Every `ExecuteTool` RPC is subject to a deadline enforced at two layers:

1. The Go kernel sets a gRPC deadline of `manifest.max_runtime_ms` milliseconds on the outbound request.
2. The Rust sidecar independently enforces this timeout via a `tokio::time::timeout` wrapper.

If the deadline expires, the sandbox returns `DEADLINE_EXCEEDED` and the kernel logs `sandbox_timeout`.

### 6.3 WASM Fuel (CPU Limiting)

WASM plugins executing inside the sandbox are additionally bounded by `wasmtime` fuel metering. This prevents a malicious plugin from consuming unbounded CPU cycles:

- Default fuel budget: **100,000,000 instructions** per invocation.
- When fuel is exhausted, execution halts with `WasmExecutionError::FuelExhausted`.
- Fuel consumption is reported back to the kernel in `ToolResponse.output_json`.

---

## 7. Threat Model

### 7.1 Assets to Protect

1. **Host filesystem** — no untrusted plugin may read or write outside declared path prefixes.
2. **Host network** — no untrusted plugin may exfiltrate data to undeclared endpoints.
3. **Go kernel process space** — no untrusted code may execute in the same address space.
4. **Other agent sessions** — task isolation prevents cross-task data leakage.

### 7.2 Threats and Mitigations

| Threat | Mitigation |
|---|---|
| Unauthorized tool invocation | Manifest ACL check — tool must be pre-declared |
| Memory exhaustion via payload | 4 MB gRPC message limit + cgroup v2 memory.max |
| CPU exhaustion via WASM loop | wasmtime fuel metering + execution timeout |
| Privilege escalation via subprocess | `CLONE_NEWPID` namespace isolates PID tree |
| Network exfiltration | `CLONE_NEWNET` namespace drops all network interfaces |
| Filesystem traversal | `CLONE_NEWNS` mount namespace + path allowlisting |
| Stale socket hijack | Socket file recreated on every boot; peer UID verified |
| Replay attack on IPC | gRPC deadline + Phase 2 mTLS certificate binding |
| Malformed protobuf DoS | Protobuf unknown-field dropping + message size limit |

### 7.3 Out of Scope (V1.0)

The following are explicitly deferred and **not** covered by this specification:
- Side-channel attacks (Spectre/Meltdown) — requires hypervisor-level isolation.
- SELinux/AppArmor mandatory access control — planned for Phase 3.
- Cryptographically signed tool manifests — planned for Phase 2 (mTLS epoch).

---

## 8. Audit Logging

Every event that crosses the IPC boundary is logged in the structured OpenTelemetry JSON format used throughout AetherCore. The minimum required fields per event:

```json
{
  "timestamp": "...",
  "level": "INFO",
  "msg": "ipc_tool_dispatched",
  "tool_name": "fetch_url",
  "caller_pid": 12345,
  "sandbox_pid": 12346,
  "outcome": "success",
  "duration_ms": 42,
  "component": "sandbox_dispatcher"
}
```

All security-relevant events (`capability_denied`, `sandbox_timeout`, `peer_uid_rejected`, `cgroup_oom`) are logged at `ERROR` level and must never be suppressed.

---

## 9. Security Contacts

For vulnerabilities found in the IPC boundary, follow the responsible disclosure process described in `SECURITY.md`. The IPC boundary is considered a **critical security boundary** and all issues are treated with the highest severity.
