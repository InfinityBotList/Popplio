import pydantic
import requests
import typing
import pathlib
from __libast import Struct, parse_file_structs

# Schema
class Schema(pydantic.BaseModel):
    table_name: str
    column_name: str
    type: str
    nullable: bool
    array: bool
    default_sql: str | None = None
    default_val: typing.Any | None = None
    secret: bool

class SchemaList(pydantic.BaseModel):
    schemas: list[Schema]

print("Fetching CI seed")

ci_seed = requests.get("https://cdn.infinitybots.gg/dev/seed-ci.json")

if ci_seed.status_code != 200:
    print("Failed to fetch CI seed")
    exit(1)

ci_data = SchemaList(schemas=ci_seed.json())

# Loop over all files in thw types folder recursively
structs: dict[str, Struct] = {}

for path in pathlib.Path("types").rglob("*"):
    print("Validating DB fields in:", path)

    # Skip directories
    if path.is_dir():
        continue

    # Open the file
    with open(path, "r") as f:
        lines = f.read().split("\n")

    # Loop over all lines, and find all structs, add the struct and its fields to a dict
    file_structs = parse_file_structs(lines)

    if not file_structs:
        continue

    # Add all file_structs to the structs dict
    for struct_name, struct in file_structs.items():
        structs[struct_name] = struct

print(structs)