package security

type ToolManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WasmCode    []byte `json:"wasm_code,omitempty"`
}
