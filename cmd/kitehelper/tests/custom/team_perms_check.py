import os

# Some constants
FILE = "teams/perms.go"
START_ENUM = "const ("
ENUM_NAME = "TeamPermission"
END_ENUM = ")"
START_DETAILS_MAP = "var TeamPermDetails = []PermDetailMap{"
END_DETAILS_MAP = "}"
ALLOWED_GROUPS = ["Bot", "Server", "Team", "Common", "undefined"]

# Read the file
with open(FILE, "r") as f:
    lines = f.read().split("\n")

enum: list[str] = []
details_map: list[str] = []

in_enum = False
in_details_map = False

for line in lines:
    # Ignore comments and empty lines
    line = line.replace("\t", "").replace(" ", "")
    if line.startswith("//") or line.startswith("/*") or not line:
        continue

    # Handle case where comments are in the middle of the line
    if "//" in line and not line.startswith("//"):
        line = line.split("//")[0].replace(" ", "")

    if line.startswith(START_ENUM.replace(" ", "")) and not in_details_map and not in_enum:
        in_enum = True

    elif line.endswith(END_ENUM.replace(" ", "")) and in_enum and not in_details_map:
        in_enum = False

    elif line.startswith(START_DETAILS_MAP.replace(" ", "")) and not in_enum and not in_details_map:
        in_details_map = True

    elif line.endswith(END_DETAILS_MAP.replace(" ", "")) and in_details_map and not in_enum:
        in_details_map = False

    elif in_enum:
        # Remove prefix, Split by enum name and get first part to get the enum name
        if not line.startswith(ENUM_NAME):
            print("FATAL: Enum must start with enum name: ", line)
            exit(1)

        enum.append(ENUM_NAME + line.replace(ENUM_NAME, "", 1).split(ENUM_NAME)[0])

    elif in_details_map:
        flag = False
        for g in ALLOWED_GROUPS:
            # Get the group (splitting by comma and removing the trailing },
            group = line.split(",")[3].replace("}", "").replace('"', "")
            if group.endswith(g):
                flag = True
                break
        
        line = line.split(",")[0].replace("{", "")

        if not flag:
            print(f"FATAL: Enum {line} must use the following group: ", ALLOWED_GROUPS)
            exit(1)

        details_map.append(line)

print("Enum: ", enum)
print("Details Map: ", details_map)

# Check if enum and details map are the same
if enum != details_map:
    print("FATAL: Enum and details map are not the same")

    add_to_enum = []
    change_line_no_enum = []
    add_to_details_map = []
    change_line_no_details_map = []

    for e in enum:
        if e not in details_map:
            add_to_details_map.append(e)
        else: # Dont prompt uneeded line no changes
            # Check if we need to change the line number
            try:
                enum_line_no = enum.index(e)
                details_map_line_no = details_map.index(e)
                if enum_line_no != details_map_line_no:
                    change_line_no_details_map.append([e, enum_line_no, details_map_line_no])
            except ValueError:
                continue
    
    for d in details_map:
        if d not in enum:
            add_to_enum.append(d)
        else: # Dont prompt uneeded line no changes
            # Check if we need to change the line number
            try:
                enum_line_no = enum.index(d)
                details_map_line_no = details_map.index(d)
                if enum_line_no != details_map_line_no:
                    change_line_no_enum.append([d, enum_line_no, details_map_line_no])
            except ValueError:
                continue
    
    # Send diff
    if add_to_enum:
        for e in add_to_enum:
            print(f"Add {e} to enum")
            exit(1)
    if add_to_details_map:
        for d in add_to_details_map:
            print(f"Add {d} to details map")

    if not add_to_enum and not add_to_details_map: # Dont prompt uneeded changes
        if change_line_no_enum:
            for c in change_line_no_enum:
                print(f"HINT: Change pos of {c[0]} in enum from {c[1]} to {c[2]} => {c[1] - c[2]} pos's")
        if change_line_no_details_map:
            for c in change_line_no_details_map:
                print(f"HINT: Change pos of {c[0]} in details map from {c[1]} to {c[2]} => {c[1] - c[2]} pos's")

    exit(1)