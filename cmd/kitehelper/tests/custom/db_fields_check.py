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

    def find(self, table_name: str, column_name: str) -> Schema | None:
        for schema in self.schemas:
            if schema.table_name == table_name and schema.column_name == column_name:
                return schema

        return None

print("Fetching CI seed...")

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

    # Add all file_structs to the structs dict if the 'table' attr is set on it
    for struct_name, struct in file_structs.items():
        if struct.attrs.get("table"):
            structs[struct_name] = struct

print("Checking structs", ", ".join(structs.keys()))

print("Check 1: Check fields to ensure they actually exist on db")

for struct_name, struct in structs.items():
    for field in struct.fields:
        if field.internal():
            # One thing to check is for a comment
            if not field.comment:
                print(f"FATAL: Field {struct_name}.{field.name} is internal but has no comment as to why")
                exit(1)

            continue

        col_name = field.tags.get("db")

        if not col_name or col_name == "-":
            print(f"FATAL: Field {struct_name}.{field.name} has no db tag. If it is internal, mark it using ci:\"internal\"")
            exit(1)

        db_col = ci_data.find(struct.attrs["table"], col_name)

        if not db_col:
            print(f"FATAL: Field {struct_name}.{field.name} with column {struct.attrs['table']}.{col_name} does not exist in the DB")
            exit(1)

print("Check 2: Check db fields to look for missing fields")

for struct_name, struct in structs.items():
    if struct.attrs.get("unfilled") == "1":
        continue

    field_db_col_names = list(map(lambda x: x.tags["db"], filter(lambda x: not x.internal(), struct.fields)))
    for ci_schema in ci_data.schemas:
        if struct.attrs["table"] != ci_schema.table_name:
            continue
        
        found = ci_schema.column_name in field_db_col_names

        if not ci_schema.secret and not found:
            print(f"FATAL: {ci_schema.table_name}.{ci_schema.column_name} is missing from {struct_name}")
            exit(1)
        
        if ci_schema.secret and found:
            print(f"FATAL: {ci_schema.table_name}.{ci_schema.column_name} is marked as secret but is in {struct_name}")
            exit(1)