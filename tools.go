package q

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadFileToolImpl implements file reading functionality
type ReadFileToolImpl struct{}

func NewReadFileTool() *ReadFileToolImpl {
	return &ReadFileToolImpl{}
}

func (t *ReadFileToolImpl) Name() string {
	return "read_file"
}

func (t *ReadFileToolImpl) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileToolImpl) Schema() Schema {
	return Schema{
		Type: "object",
		Properties: map[string]SchemaField{
			"path": {
				Type:        "string",
				Description: "Path to the file to read",
			},
		},
		Required: []string{"path"},
	}
}

func (t *ReadFileToolImpl) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	var params struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return &ToolResult{Error: "invalid input parameters"}, nil
	}

	if params.Path == "" {
		return &ToolResult{Error: "path parameter is required"}, nil
	}

	content, err := os.ReadFile(params.Path)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to read file: %v", err)}, nil
	}

	return &ToolResult{
		Content: string(content),
		Metadata: map[string]any{
			"path": params.Path,
			"size": len(content),
		},
	}, nil
}

// WriteFileToolImpl implements file writing functionality
type WriteFileToolImpl struct{}

func NewWriteFileTool() *WriteFileToolImpl {
	return &WriteFileToolImpl{}
}

func (t *WriteFileToolImpl) Name() string {
	return "write_file"
}

func (t *WriteFileToolImpl) Description() string {
	return "Write content to a file"
}

func (t *WriteFileToolImpl) Schema() Schema {
	return Schema{
		Type: "object",
		Properties: map[string]SchemaField{
			"path": {
				Type:        "string",
				Description: "Path to the file to write",
			},
			"content": {
				Type:        "string",
				Description: "Content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	}
}

func (t *WriteFileToolImpl) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return &ToolResult{Error: "invalid input parameters"}, nil
	}

	if params.Path == "" {
		return &ToolResult{Error: "path parameter is required"}, nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(params.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to create directory: %v", err)}, nil
	}

	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to write file: %v", err)}, nil
	}

	return &ToolResult{
		Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.Path),
		Metadata: map[string]any{
			"path": params.Path,
			"size": len(params.Content),
		},
	}, nil
}

// ListFilesToolImpl implements directory listing functionality
type ListFilesToolImpl struct{}

func NewListFilesTool() *ListFilesToolImpl {
	return &ListFilesToolImpl{}
}

func (t *ListFilesToolImpl) Name() string {
	return "list_files"
}

func (t *ListFilesToolImpl) Description() string {
	return "List files and directories in a given path"
}

func (t *ListFilesToolImpl) Schema() Schema {
	return Schema{
		Type: "object",
		Properties: map[string]SchemaField{
			"path": {
				Type:        "string",
				Description: "Path to list (defaults to current directory)",
				Default:     ".",
			},
		},
	}
}

func (t *ListFilesToolImpl) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	var params struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return &ToolResult{Error: "invalid input parameters"}, nil
	}

	if params.Path == "" {
		params.Path = "."
	}

	entries, err := os.ReadDir(params.Path)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to read directory: %v", err)}, nil
	}

	var files []string
	var dirs []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name()+"/")
		} else {
			files = append(files, entry.Name())
		}
	}

	result := fmt.Sprintf("Directory: %s\n", params.Path)
	if len(dirs) > 0 {
		result += "Directories:\n"
		for _, dir := range dirs {
			result += fmt.Sprintf("  %s\n", dir)
		}
	}
	if len(files) > 0 {
		result += "Files:\n"
		for _, file := range files {
			result += fmt.Sprintf("  %s\n", file)
		}
	}

	return &ToolResult{
		Content: result,
		Metadata: map[string]any{
			"path":       params.Path,
			"file_count": len(files),
			"dir_count":  len(dirs),
		},
	}, nil
}

// ExecToolImpl implements command execution functionality
type ExecToolImpl struct{}

func NewExecTool() *ExecToolImpl {
	return &ExecToolImpl{}
}

func (t *ExecToolImpl) Name() string {
	return "exec"
}

func (t *ExecToolImpl) Description() string {
	return "Execute a shell command"
}

func (t *ExecToolImpl) Schema() Schema {
	return Schema{
		Type: "object",
		Properties: map[string]SchemaField{
			"command": {
				Type:        "string",
				Description: "Command to execute",
			},
			"args": {
				Type:        "array",
				Description: "Command arguments",
			},
			"working_dir": {
				Type:        "string",
				Description: "Working directory for the command",
			},
		},
		Required: []string{"command"},
	}
}

func (t *ExecToolImpl) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	var params struct {
		Command    string   `json:"command"`
		Args       []string `json:"args"`
		WorkingDir string   `json:"working_dir"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return &ToolResult{Error: "invalid input parameters"}, nil
	}

	if params.Command == "" {
		return &ToolResult{Error: "command parameter is required"}, nil
	}

	// For security, this is a simplified implementation
	// In production, you'd want to restrict allowed commands
	return &ToolResult{
		Error: "command execution disabled for security",
	}, nil
}
