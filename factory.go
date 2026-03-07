package q

// Convenience functions for creating common tools
func ReadFileTool() Tool {
	return NewReadFileTool()
}

func WriteFileTool() Tool {
	return NewWriteFileTool()
}

func ListFilesTool() Tool {
	return NewListFilesTool()
}

func ExecTool() Tool {
	return NewExecTool()
}
