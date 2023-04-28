import pydantic
import shlex

# Struct data
class StructField(pydantic.BaseModel):
    name: str
    type: str
    tags: dict[str, str]

class Struct(pydantic.BaseModel):
    attrs: list[str]
    fields: list[StructField]

def parse_go_struct_field(field: str) -> dict[str, str]:
    field = field.replace("`", "").replace(":", "=").split("//")[0]

    res = {}

    i = 0
    while i < len(field):
        # Get till next =
        key = ""
        value = ""
        while i < len(field) and field[i] != "=":
            key += field[i]
            i += 1
                
        # Skip = and first quotation (=" bit)
        i += 2

        # Get till next "
        while i < len(field) and field[i] != "\"":
            value += field[i]
            i += 1
        
        # Skip last quotation
        i += 1

        # Add to res
        res[key] = value

        # Skip until next non-whitespace
        while i < len(field) and field[i] == " ":
            i += 1 

    return res

def parse_file_structs(lines: list[str]) -> dict[str, Struct] | None:
    # Loop over all lines, and find all structs, add the struct and its fields to a dict
    structs: dict[str, Struct] = {}

    curr_struct: str | None = None
    for line in lines:
        if not line.strip().strip("\t"):
            continue

        if line.startswith("type") and " struct" in line:
            struct_name = line.split(" ")[1]
            print("Adding struct", struct_name)
            structs[struct_name] = Struct(attrs=[], fields=[])

            # Go backwards from the current line, reading all comments starting with //@ci and add them to the struct
            for i in range(lines.index(line), 0, -1):
                if lines[i].startswith("//@ci"):
                    structs[struct_name].attrs.append(" ".join(lines[i].split(" ")[1:]))
                elif lines[i] == "" or not lines[i].startswith("//"):
                    break
            
            curr_struct = struct_name
            continue
        
        if line.endswith("}"):
            curr_struct = None
            continue

        if curr_struct is not None:
            line_split = list(filter(lambda x: x, line.replace("\t", "").split(" ")))
            sf = StructField(
                name=line_split[0].strip().strip("\t"),
                type=line_split[1].strip().strip("\t"),
                tags=parse_go_struct_field(" ".join(line_split[2:]).strip().strip("\t"))
            )
            structs[curr_struct].fields.append(sf)

            print(sf)
    
    return structs