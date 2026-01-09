# Fix Skip field reference
s/Skip:   \[\]string{"config.local"},//

# Fix Subclones to Workspaces
s/Subclones:/Workspaces:/g
s/manifest\.Subclone/manifest.WorkspaceEntry/g

# Fix error checking patterns - add 'else' before second if
/if err == nil {/{
N
N
/if !strings\.Contains(err\.Error()/{
s/\(\t*\)}\n\(\t*\)if !strings\.Contains(err\.Error()/\1} else if !strings.Contains(err.Error()/
}
}
