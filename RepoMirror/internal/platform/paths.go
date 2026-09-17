package platform

func buildRootedPath(buffer []byte, root string, relPath string) []byte {
	if root == "" {
		return appendNativeRelativePath(buffer[:0], relPath)
	}
	buffer = append(buffer[:0], root...)
	if !isPathSeparator(root[len(root)-1]) {
		buffer = append(buffer, '\\')
	}
	return appendNativeRelativePath(buffer, relPath)
}

func appendNativeRelativePath(buffer []byte, relPath string) []byte {
	for index := 0; index < len(relPath); index++ {
		if relPath[index] == '/' {
			buffer = append(buffer, '\\')
			continue
		}
		buffer = append(buffer, relPath[index])
	}
	return buffer
}

func isPathSeparator(char byte) bool {
	return char == '/' || char == '\\'
}
