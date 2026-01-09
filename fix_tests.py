#!/usr/bin/env python3
import re

# Read the file
with open('cmd/cmd_test.go', 'r') as f:
    lines = f.readlines()

# Fix 1 & 2: Simple replacements on each line
for i in range(len(lines)):
    # Remove Skip field
    if 'Skip:   []string{"config.local"},' in lines[i]:
        lines[i] = ''
        continue

    # Update comment
    lines[i] = lines[i].replace('// Create manifest with ignore/skip patterns',
                                '// Create manifest with ignore patterns (Skip field removed in v0.1.0)')

    # Replace Subclones with Workspaces
    lines[i] = lines[i].replace('Subclones:', 'Workspaces:')
    lines[i] = lines[i].replace('manifest.Subclone', 'manifest.WorkspaceEntry')

# Fix 3: Fix error checking patterns
# Change: \t}\n\tif !strings.Contains... to: \t} else if !strings.Contains...
fixed_lines = []
i = 0
while i < len(lines):
    # Check if this is a closing brace followed by "if !strings.Contains(err.Error()"
    if i + 1 < len(lines) and re.match(r'^\s+\}$', lines[i].rstrip()):
        # Check if next line is "if !strings.Contains(err.Error()"
        if re.match(r'^\s+if !strings\.Contains\(err\.Error\(\)', lines[i + 1]):
            # Get the indentation and rest of next line
            next_line = lines[i + 1]
            match = re.match(r'^(\s+)if (!strings\.Contains\(err\.Error\(\).*)', next_line)
            if match:
                indent = match.group(1)
                condition = match.group(2)
                # Append brace line with "} else if..."
                fixed_lines.append(lines[i].rstrip() + ' else if ' + condition)
                # Skip the next line (the original "if !strings.Contains...")
                i += 2
                continue

    fixed_lines.append(lines[i])
    i += 1

# Write back
with open('cmd/cmd_test.go', 'w') as f:
    f.writelines(fixed_lines)

print("File fixed successfully")
