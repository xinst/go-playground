import config from '~/services/config';
import type { WorkspaceState } from './types';

const CONFIG_KEY = 'workspace.state';

const defaultFile = `
package main

import (
\t"fmt"
)

func main() {
\tfmt.Println("Hello, World!")
}
`.trimStart()

const defaultWorkspace: WorkspaceState = {
  selectedFile: 'main.go',
  files: {
    'main.go': defaultFile,
  }
}

export const saveWorkspaceState = (state: WorkspaceState) => {
  const sanitized = sanitizeState(state);
  if (!sanitized.files) {
    // Truncate saved state if workspace is empty.
    config.delete(CONFIG_KEY);
    return;
  }

  config.setObject(CONFIG_KEY, sanitized);
};

export const loadWorkspaceState = (): WorkspaceState => (
  sanitizeState(config.getObject(CONFIG_KEY, defaultWorkspace))
);

const sanitizeState = (state: WorkspaceState) => {
  // Skip current snippet URL.
  const { selectedFile, files } = state;

  if (!files) {
    // Save defaults if ws is empty.
    return defaultWorkspace;
  }

  return { selectedFile, files };
}

