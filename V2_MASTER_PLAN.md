# AetherCore v2.0 Complete Enterprise-Grade Technical Master Plan

_(Migrated from V1_LAUNCH_PLAN.md to explicitly target Enterprise Security & Multi-Agent orchestration)_

## Architecture Overview — What We're Building

```
┌─────────────────────────────────────────────────────────────────┐
│                    AetherCore v2.0 Full Stack                   │
├─────────────────────────────────────────────────────────────────┤
│  Layer 5: Clients                                               │
│  Web Dashboard | REST API | Discord | Telegram | Slack | WA     │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: Enterprise                                            │
│  RBAC | Audit Chain | Budget Manager | Rate Limiter | SSO       │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Intelligence                                          │
│  Multi-Agent Orchestrator | Memory Engine | LLM Router          │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: Security Kernel                                       │
│  Prompt Guard | Tool Verifier | Rust Sandbox | Namespace ISO    │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: Core Kernel (SACRED — never touch)                    │
│  Event Loop | IPC | Mesh | mTLS | Telemetry                     │
└─────────────────────────────────────────────────────────────────┘
```

_Rule: Higher layers depend on lower layers. Lower layers never depend on higher layers._

## Phase 0 — Foundation Fix (Week 1–2)

0.1 **ReAct Loop Fix**: The event loop must feed tool responses back into the LLM context to create an actual autonomous agent, instead of a simple one-shot wrapper.
0.2 **Rust Sandbox to Production**: Replace the stub sandbox with actual `manifest.toml` capability enforcement, Linux `cgroups` and `namespaces` resource isolation via `nix`, and a deterministic `wasmtime` execution engine.
0.3 **E2E Validation**: Create a bash integration script ensuring <15MB memory limits and robust prevention of sandbox breakouts.

## Phase 1 — Security Core (Week 3–5)

1.1 **Prompt Injection Defense**: Kernel-level protection blocking prompt leakage and overrides (DAN/STAN), running both on user inputs and crucially, on tool outputs.
1.2 **Cryptographic Tool Verification**: `Ed25519` signature checks against official keys before allowing tools to be registered or compiled.
1.3 **Immutable Audit Chain**: Append-only execution logs hashed linearly to prevent post-execution tampering.

## Phase 2 — Intelligence Layer (Week 6–8)

2.1 **Multi-LLM Router**: Fallback chains bridging OpenAI, Anthropic, DeepSeek, and Local Ollama instances. Incorporates cost-awareness and privacy-mode.
2.2 **Persistent Memory**: SQLite-driven episodic memory consolidated by background routines.
2.3 **Seven Core Skills**: Builtin WASM tools for Web Search, File Management, Code Running, etc., bound by strict capability manifests.

## Phase 3 — Multi-Agent (Week 9–11)

3.1 **Agent Orchestrator**: Supervisor, Executor, Critic, and Planner roles managing concurrent sub-tasks through DAG dependency representations.
3.2 **Mesh Propagation**: Expanding tasks securely across `mTLS` validated peer nodes dynamically.

## Phase 4 — Enterprise (Week 12–14)

4.1 **RBAC**: Administrator, Developer, Operator, Guest hierarchical access.
4.2 **Budget Manager**: Soft and hard token spend limits tracked across actors.
4.3 **Observability**: Prometheus metrics.
4.4 **Kubernetes Support**: Helm deployment templates.

## Phase 5 — Dashboard (Week 15–17)

5.1 **Backend API**: Endpoints serving the React/Vanilla JS web dashboard.
5.2 **Frontend UI**: Builtin web portal streaming executions via WebSockets.

## Phase 6 — Gateway Expansion (Week 18–20)

6.1 **REST API Gateway**: Enterprise asynchronous workflows with Webhooks.
6.2 **Slack & WhatsApp Gateways**: Official corporate integrations.
