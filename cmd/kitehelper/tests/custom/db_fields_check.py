import pydantic
import typing
import pathlib
import json
from __libast import Struct, parse_file_structs, debug

# Schema
class Schema(pydantic.BaseModel):
    table_name: str
    column_name: str
    type: str
    nullable: bool
    array: bool
    default_sql: str | None = None
    default_val: typing.Any | None = None

class SchemaList(pydantic.BaseModel):
    schemas: list[Schema]

    def find(self, table_name: str, column_name: str) -> Schema | None:
        for schema in self.schemas:
            if schema.table_name == table_name and schema.column_name == column_name:
                return schema

        return None

print("Loading CI seed...")

with open(f"data/seed-ci.json", "r") as f:
    ci_seed = json.load(f)

ci_data = SchemaList(schemas=ci_seed)

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
        if field.tags.get("skip"):
            continue

        col_name = field.tags.get("db") or field.tags.get("pdb")
        
        if field.internal():
            if col_name not in ["", "-", None]:
                print(f"FATAL: Field {struct_name}.{field.name} is internal but has a db tag")
                exit(1)

            # One thing to check is for a comment
            if not field.comment:
                print(f"FATAL: Field {struct_name}.{field.name} is internal but has no comment as to why")
                exit(1)

            continue

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

    field_db_col_names = []

    ignore_fields = struct.attrs.get("ignore_fields", "").split("+")
    field_db_col_names.extend(ignore_fields) # Ensure we ignore these fields

    for field in struct.fields:
        if field.tags.get("skip"):
            field_db_col_names.append(field.tags.get("skip"))

        col_name = field.tags.get("db") or field.tags.get("pdb")

        if not col_name or col_name == "-":
            continue

        if not field.tags.get("json") and not field.internal():
            print(f"FATAL: Field {struct_name}.{field.name} has no json tag. If it is internal, mark it using ci:\"internal\"")
            exit(1)

        field_db_col_names.append(col_name)

    debug(field_db_col_names)

    for ci_schema in ci_data.schemas:
        if struct.attrs["table"] != ci_schema.table_name:
            continue
        
        found = ci_schema.column_name in field_db_col_names

        if not found:
            print(f"FATAL: {ci_schema.table_name}.{ci_schema.column_name} is missing from {struct_name}")
            exit(1)