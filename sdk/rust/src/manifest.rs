use serde::{Deserialize, Serialize};

/// The capability permissions a module may declare in its [`ModuleManifest`].
///
/// The AetherCore kernel validates the declared capabilities before loading a
/// module.  Exercising an undeclared capability causes the kernel to terminate
/// the module with a policy violation.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Capability {
    /// Permits outbound TCP/UDP connections.
    Network,
    /// Permits read/write access to the local filesystem (sandboxed path).
    Filesystem,
    /// Permits persistent key/value state via the kernel state store.
    State,
    /// Permits participation in the AetherCore agent mesh protocol.
    Mesh,
}

/// The declarative identity envelope every [`AetherModule`][crate::AetherModule]
/// must return from its `manifest()` method.
///
/// The kernel uses this structure to enforce capability policies, display
/// the module in `aether tool list`, and route tasks to the correct module.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModuleManifest {
    /// Unique kebab-case identifier (e.g. `"web-search"`).
    pub name: String,

    /// One-line human-readable summary of the module's purpose.
    pub description: String,

    /// Semantic version string (e.g. `"1.0.0"`).
    pub version: String,

    /// Author name or organisation.
    pub author: String,

    /// Kernel permissions this module requires.
    #[serde(default)]
    pub capabilities: Vec<Capability>,

    /// Hard deadline for a single `handle_task` call in milliseconds.
    /// The kernel injects a deadline via the WASM fuel/epoch mechanism.
    #[serde(default = "default_task_runtime_ms")]
    pub max_task_runtime_ms: u64,

    /// Advisory memory ceiling in megabytes.
    /// Enforced by the Rust Sandbox (Layer 2) WASM memory limits.
    #[serde(default = "default_memory_limit_mb")]
    pub memory_limit_mb: u64,
}

fn default_task_runtime_ms() -> u64 {
    5_000
}

fn default_memory_limit_mb() -> u64 {
    64
}
