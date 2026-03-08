//! # aether-sdk
//!
//! The AetherCore Rust SDK for building **Layer 1 Modules** — sandboxed,
//! composable units of capability that extend the AetherCore kernel.
//!
//! ## Quick Start
//!
//! Implement the [`AetherModule`] trait and export a `create_module` constructor
//! so the kernel can dynamically link your WASM module:
//!
//! ```rust
//! use aether_sdk::{AetherModule, ModuleManifest, ModuleTask, ModuleResult};
//! use async_trait::async_trait;
//!
//! struct MyModule;
//!
//! #[async_trait]
//! impl AetherModule for MyModule {
//!     fn manifest(&self) -> ModuleManifest {
//!         ModuleManifest {
//!             name: "my-module".into(),
//!             description: "An example AetherCore module".into(),
//!             version: "0.1.0".into(),
//!             author: "Jane Doe".into(),
//!             capabilities: vec![],
//!             max_task_runtime_ms: 5_000,
//!             memory_limit_mb: 64,
//!         }
//!     }
//!
//!     async fn on_start(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
//!         Ok(())
//!     }
//!
//!     async fn on_stop(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
//!         Ok(())
//!     }
//!
//!     async fn handle_task(
//!         &self,
//!         task: ModuleTask,
//!     ) -> Result<ModuleResult, Box<dyn std::error::Error + Send + Sync>> {
//!         Ok(ModuleResult::ok(task.id, "Hello from my-module!"))
//!     }
//! }
//! ```
//!
//! ## Crate Layout
//!
//! | Module | Contents |
//! |--------|----------|
//! | [`manifest`] | [`ModuleManifest`] and [`Capability`] types |
//! | [`task`]     | [`ModuleTask`] and [`ModuleResult`] types |
//! | (root)       | The [`AetherModule`] trait |

pub mod manifest;
pub mod task;

pub use manifest::{Capability, ModuleManifest};
pub use task::{ModuleResult, ModuleTask};

use async_trait::async_trait;

/// The core trait every AetherCore Rust module must implement.
///
/// The kernel calls [`on_start`][AetherModule::on_start] once when the WASM
/// sandbox is initialised, [`handle_task`][AetherModule::handle_task] for each
/// dispatched task, and [`on_stop`][AetherModule::on_stop] during graceful
/// shutdown.
///
/// Implementations must be `Send + Sync` because the kernel may invoke
/// `handle_task` concurrently from multiple worker threads.
#[async_trait]
pub trait AetherModule: Send + Sync {
    /// Returns the static metadata and capability declaration for this module.
    ///
    /// The kernel calls this once at load time to validate capabilities and
    /// register the module in the manifest catalogue.
    fn manifest(&self) -> ModuleManifest;

    /// Called once when the kernel loads the module into the sandbox.
    ///
    /// Initialise long-lived resources (HTTP clients, DB connections, caches)
    /// here.  The kernel will not dispatch tasks until this returns `Ok(())`.
    async fn on_start(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>>;

    /// Called once during graceful kernel shutdown.
    ///
    /// Release all held resources.  The kernel enforces a shutdown deadline
    /// derived from the WASM epoch mechanism; work performed after the deadline
    /// is silently discarded.
    async fn on_stop(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>>;

    /// Processes a single task dispatched by the kernel.
    ///
    /// This method **must be safe for concurrent calls** — multiple tasks may
    /// be in-flight simultaneously when the kernel runs with a thread-pool.
    /// Use interior mutability (`Arc<Mutex<_>>`) for any shared mutable state.
    async fn handle_task(
        &self,
        task: ModuleTask,
    ) -> Result<ModuleResult, Box<dyn std::error::Error + Send + Sync>>;
}

#[cfg(test)]
mod tests {
    use super::*;

    struct EchoModule;

    #[async_trait]
    impl AetherModule for EchoModule {
        fn manifest(&self) -> ModuleManifest {
            ModuleManifest {
                name: "echo".into(),
                description: "Echoes task input back as output".into(),
                version: "0.1.0".into(),
                author: "AetherCore".into(),
                capabilities: vec![],
                max_task_runtime_ms: 1_000,
                memory_limit_mb: 16,
            }
        }

        async fn on_start(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
            Ok(())
        }

        async fn on_stop(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
            Ok(())
        }

        async fn handle_task(
            &self,
            task: ModuleTask,
        ) -> Result<ModuleResult, Box<dyn std::error::Error + Send + Sync>> {
            Ok(ModuleResult::ok(&task.id, &task.input))
        }
    }

    #[tokio::test]
    async fn echo_module_round_trips_task() {
        let mut m = EchoModule;
        m.on_start().await.unwrap();

        let task = ModuleTask {
            id: "t1".into(),
            input: "hello".into(),
            metadata: Default::default(),
        };
        let result = m.handle_task(task).await.unwrap();
        assert_eq!(result.task_id, "t1");
        assert_eq!(result.output, "hello");

        m.on_stop().await.unwrap();
    }

    #[test]
    fn manifest_serialises_to_json() {
        let m = EchoModule;
        let json = serde_json::to_string(&m.manifest()).unwrap();
        assert!(json.contains("\"name\":\"echo\""));
        assert!(json.contains("\"version\":\"0.1.0\""));
    }
}
