#![allow(dead_code)]

use ed25519_dalek::{Signature, Verifier, VerifyingKey as PublicKey};
use serde::{Deserialize, Serialize};

#[derive(Debug)]
pub enum ManifestError {
    SerializationFailed,
    InvalidSignatureEncoding,
    InvalidSignature,
    SignatureVerificationFailed,
}

impl std::fmt::Display for ManifestError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{:?}", self)
    }
}

impl std::error::Error for ManifestError {}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct Manifest {
    pub sandbox: SandboxConfig,
    pub tools: Vec<ToolManifest>,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct SandboxConfig {
    pub strict_mode: bool,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct ToolManifest {
    pub name:         String,
    pub version:      String,
    pub publisher:    String,
    pub capabilities: Capabilities,
    pub signature:    String, // Ed25519 over canonical JSON of everything above
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct Capabilities {
    pub filesystem:    Vec<FsRule>,      // [{path: "/tmp/out", mode: "rw"}]
    pub network:       NetworkPolicy,    // Deny | AllowList(Vec<String>)
    pub env_vars:      Vec<String>,      // explicit whitelist
    pub max_memory_mb: u64,
    pub max_cpu_ms:    u64,
    pub wasm_only:     bool,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct FsRule {
    pub path: String,
    pub mode: String, // e.g. "ro" or "rw"
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
#[serde(untagged)]
pub enum NetworkPolicy {
    Deny(String), // "Deny"
    AllowList(Vec<String>), // domain whitelist
}

// CanonicalManifest removes the signature field to create the digest payload.
#[derive(Serialize)]
struct CanonicalManifest<'a> {
    name:         &'a str,
    version:      &'a str,
    publisher:    &'a str,
    capabilities: &'a Capabilities,
}

impl<'a> From<&'a ToolManifest> for CanonicalManifest<'a> {
    fn from(m: &'a ToolManifest) -> Self {
        Self {
            name: &m.name,
            version: &m.version,
            publisher: &m.publisher,
            capabilities: &m.capabilities,
        }
    }
}

impl ToolManifest {
    pub fn verify(&self, trusted_pubkey: &PublicKey) -> Result<(), ManifestError> {
        // Canonical JSON serialize (sorted keys, no whitespace) using a canonicalizer or just basic serde_json
        // In a true production system, you'd use a deterministic JSON serializer like olpc-cjson or jcs.
        // For Phase 0, standard serde_json to_vec serves as the foundation.
        let payload = serde_json::to_vec(&CanonicalManifest::from(self))
            .map_err(|_| ManifestError::SerializationFailed)?;

        let sig_bytes = hex::decode(&self.signature)
            .map_err(|_| ManifestError::InvalidSignatureEncoding)?;

        let sig_array: [u8; 64] = sig_bytes.as_slice().try_into()
            .map_err(|_| ManifestError::InvalidSignature)?;
        
        let signature = Signature::from_bytes(&sig_array);

        trusted_pubkey
            .verify(&payload, &signature)
            .map_err(|_| ManifestError::SignatureVerificationFailed)?;

        Ok(())
    }

    pub fn allows_fs_path(&self, path: &str, write: bool) -> bool {
        for rule in &self.capabilities.filesystem {
            if path.starts_with(&rule.path) {
                if write { return rule.mode.contains('w'); }
                return true;
            }
        }
        false
    }

    pub fn allows_network(&self, domain: &str) -> bool {
        match &self.capabilities.network {
            NetworkPolicy::Deny(_) => false,
            NetworkPolicy::AllowList(domains) => {
                domains.iter().any(|d| domain == d || domain.ends_with(&format!(".{}", d)))
            }
        }
    }
}

impl Manifest {
    pub fn load<P: AsRef<std::path::Path>>(path: P) -> Result<Self, Box<dyn std::error::Error>> {
        let content = std::fs::read_to_string(path)?;
        let manifest: Manifest = toml::from_str(&content)?;
        Ok(manifest)
    }
}
