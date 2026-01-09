#!/usr/bin/env python3
import re

# Read the file
with open('cmd/cmd_test.go', 'r') as f:
    lines = f.readlines()

# Fix 1: Remove Skip field
# Fix 2: Update comment
# Fix 3: Replace Subclones with Workspaces
for i in range(len(lines)):
    if 'Skip:   []string{"config.local"},' in lines[i]:
        lines[i] = ''
        continue

    lines[i] = lines[i].replace('// Create manifest with ignore/skip patterns',
                                '// Create manifest with ignore patterns (Skip field removed in v0.1.0)')
    lines[i] = lines[i].replace('Subclones:', 'Workspaces:')
    lines[i] = lines[i].replace('manifest.Subclone', 'manifest.WorkspaceEntry')

# Fix 4: Comment out push-related tests
in_push_test = False
comment_depth = 0
push_test_names = [
    'TestRunPush',
    'TestPushSubclone',
    'TestPushAllNoSubclones',
    'TestRunPushNotInGitRepo',
    'TestPushSubcloneWithPushError',
    'TestPushAllSkipsNotCloned',
    'TestPushAllWithHasChangesError',
    'TestPushAllWithPushError',
    'TestPushWithManifestLoadError'
]

fixed_lines = []
i = 0
while i < len(lines):
    line = lines[i]

    # Check if this is the start of a push test
    for test_name in push_test_names:
        if f'func {test_name}(t *testing.T)' in line:
            # Add comment before function
            fixed_lines.append(f'// {test_name} removed - push functionality was removed in v0.1.0\n')
            fixed_lines.append('/*\n')
            in_push_test = True
            comment_depth = 0
            break

    # If in push test, track braces
    if in_push_test:
        fixed_lines.append(line)
        comment_depth += line.count('{') - line.count('}')
        if comment_depth == 0:
            fixed_lines.append('*/\n')
            fixed_lines.append('\n')
            in_push_test = False
            i += 1
            continue
    else:
        fixed_lines.append(line)

    i += 1

# Fix 5: Fix error checking patterns
# Change: \t}\n\tif !strings.Contains... to: \t} else if !strings.Contains...
final_lines = []
i = 0
while i < len(fixed_lines):
    if i + 1 < len(fixed_lines) and re.match(r'^\s+\}$', fixed_lines[i].rstrip()):
        if re.match(r'^\s+if !strings\.Contains\(err\.Error\(\)', fixed_lines[i + 1]):
            next_line = fixed_lines[i + 1]
            match = re.match(r'^(\s+)if (!strings\.Contains\(err\.Error\(\).*)', next_line)
            if match:
                condition = match.group(2)
                final_lines.append(fixed_lines[i].rstrip() + ' else if ' + condition)
                i += 2
                continue

    final_lines.append(fixed_lines[i])
    i += 1

# Write back
with open('cmd/cmd_test.go', 'w') as f:
    f.writelines(final_lines)

print("All fixes applied successfully")
