use std::collections::HashMap;

use serde::{Deserialize, Serialize};

/// A single unit of work dispatched from the kernel to a module.
///
/// Modules receive `ModuleTask` values in their
/// [`handle_task`][crate::AetherModule::handle_task] implementation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModuleTask {
    /// Opaque unique identifier assigned by the kernel.
    pub id: String,

    /// The plain-text or JSON instruction string for this task.
    pub input: String,

    /// Arbitrary key/value metadata forwarded by the kernel (e.g. trace IDs).
    #[serde(default)]
    pub metadata: HashMap<String, String>,
}

/// The result produced by a module after processing a [`ModuleTask`].
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModuleResult {
    /// Must match [`ModuleTask::id`] of the task that produced this result.
    pub task_id: String,

    /// The plain-text or JSON output produced by the module.
    pub output: String,

    /// Arbitrary key/value metadata the module wishes to surface to the kernel.
    #[serde(default)]
    pub metadata: HashMap<String, String>,
}

impl ModuleResult {
    /// Convenience constructor for a successful result with no metadata.
    pub fn ok(task_id: impl Into<String>, output: impl Into<String>) -> Self {
        Self {
            task_id: task_id.into(),
            output: output.into(),
            metadata: HashMap::new(),
        }
    }
}
