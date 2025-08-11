package tools

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *Registry) {
	// Basic tools
	registry.Register(&ShellTool{})
	registry.Register(&CurrentTimeTool{})
	
	// File operations
	registry.Register(&FileReadTool{})
	registry.Register(&FileEditTool{})
	registry.Register(&FileCreationTool{})
	registry.Register(&FileReplaceLinesTool{})
	registry.Register(&FileSearchReplaceTool{})
	registry.Register(&FileInsertTool{})
	registry.Register(&FileManageTool{})
	
	// Directory operations
	registry.Register(&DirectoryManageTool{})
	
	// Development tools
	registry.Register(&CodeFormatterTool{})
	
	// Data tools
	registry.Register(&DataEditTool{})
	registry.Register(&DataProcessTool{})
	
	// Network tools
	registry.Register(&HttpRequestTool{})
	
	// System tools (to be implemented in separate files)
	// registry.Register(&FileDiffTool{})
	// registry.Register(&EnvManageTool{})
	// registry.Register(&ProcessManageTool{})
}