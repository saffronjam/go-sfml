import json
import os
import re

# Get from environment variables or error
AST_DIR = os.getenv('AST_DIR')
if not AST_DIR:
    raise ValueError(
        "Environment variable AST_DIR is not set. Please set it to the directory containing the AST JSON files.")
if not os.path.isdir(AST_DIR):
    raise ValueError(f"AST_DIR '{AST_DIR}' is not a valid directory.")

# Output directories
JSON_DIR = os.getenv('JSON_DIR')
if not JSON_DIR:
    raise ValueError(
        "Environment variable JSON_DIR is not set. Please set it to the directory where you want to save the output files.")
if not os.path.isdir(JSON_DIR):
    raise ValueError(f"JSON_DIR '{JSON_DIR}' is not a valid directory.")

# Output files
TYPES_OUT = os.path.join(JSON_DIR, 'types.json')
FUNCS_OUT = os.path.join(JSON_DIR, 'functions.json')

# Regex patterns for filtering what should be passed
# 1. Starts with sf
name_reqex = re.compile(r'^sf[A-Z][a-zA-Z0-9_]*')

# Skip Network and Audio files
skip_files_regex = re.compile(r'^(SFML_Network|SFML_Audio)')


def should_include(name):
    """Check if a name matches the required pattern."""
    return name_reqex.match(name)


def extract_types(ast_node):
    types = []

    if isinstance(ast_node, dict):
        kind = ast_node.get('kind')

        if kind == 'TypedefDecl':
            name = ast_node.get('name')
            if name and should_include(name):
                children_ids = set()
                for child in ast_node.get('inner', []):
                    if "ownedTagDecl" in child:
                        children_ids.add(child['ownedTagDecl'].get('id'))

                types.append({
                    'id': ast_node.get('id'),
                    'kind': kind,
                    'name': name,
                    'type': ast_node.get('type', {}).get('qualType', 'unknown'),
                    'children_ids': children_ids
                })

        if kind in ['RecordDecl', 'EnumDecl']:
            type_info = {
                'id': ast_node.get('id'),
                'kind': kind,
            }
            if kind == 'RecordDecl':
                fields = []
                for child in ast_node.get('inner', []):
                    if child.get('kind') == 'FieldDecl':
                        fields.append({
                            'name': child.get('name'),
                            'type': child.get('type', {}).get('qualType')
                        })
                type_info['fields'] = fields

            if kind == 'EnumDecl':
                enumerators = []
                for child in ast_node.get('inner', []):
                    if child.get('kind') == 'EnumConstantDecl':
                        enumerators.append({"name": child.get('name')})
                type_info['enumerators'] = enumerators
            types.append(type_info)

        for child in ast_node.get('inner', []):
            types.extend(extract_types(child))

    elif isinstance(ast_node, list):
        for item in ast_node:
            types.extend(extract_types(item))

    return types


def extract_functions(ast_node):
    functions = []

    if isinstance(ast_node, dict):
        if ast_node.get('kind') == 'FunctionDecl':
            fn = {
                'name': ast_node.get('name'),
                'return_type': ast_node.get('type', {}).get('qualType', 'void').split('(')[0],
                'parameters': []
            }

            if not should_include(fn['name']):
                return functions

            for child in ast_node.get('inner', []):
                if child.get('kind') == 'ParmVarDecl':
                    fn['parameters'].append({
                        'name': child.get('name'),
                        'type': child.get('type', {}).get('qualType', 'unknown')
                    })

            functions.append(fn)

        for child in ast_node.get('inner', []):
            functions.extend(extract_functions(child))

    elif isinstance(ast_node, list):
        for item in ast_node:
            functions.extend(extract_functions(item))

    return functions


# --- Main processing ---
all_types = []
all_functions = []

for filename in os.listdir(AST_DIR):
    if skip_files_regex.match(filename):
        print(f"Skipping {filename} as it matches the skip pattern.")
        continue

    if filename.endswith('.json'):
        with open(os.path.join(AST_DIR, filename)) as f:
            print(f"🔍 Processing {filename}...")
            ast = json.load(f)
            all_types.extend(extract_types(ast))
            all_functions.extend(extract_functions(ast))

all_sfVector2u = [t for t in all_types if t.get('name') == 'sfVector2u']

# --- Deduplicate types by name ---
unique_types = {}
for t in all_types:

    # EnumDecl and RecordDecl does not have a name, match it by childrenIds
    if t['kind'] == 'EnumDecl' or t['kind'] == 'RecordDecl':
        # Find in all_types a TypeDefDecl with t.id in its childrenIds
        type_decl = next((
            td for td in all_types
            if td['kind'] == 'TypedefDecl' and t['id'] in td.get('children_ids', set())
        ), None)
        t['name'] = t.get('name', type_decl.get('name') if type_decl else None)

    name = t.get('name')
    if not name:
        continue

    if name in unique_types:
        # If a type with the same name already exists, merge fields or enumerators, overwrite
        existing = unique_types[name]

        if t['kind'] == 'EnumDecl':
            existing.setdefault('enumerators', []).extend(t.get('enumerators', []))
            # Ensure unique enumerators
            existing['enumerators'] = list({e['name']: e for e in existing['enumerators']}.values())
            existing.setdefault('type', 'enum')
        elif t['kind'] == 'RecordDecl':
            existing.setdefault('fields', []).extend(t.get('fields', []))
            # Ensure unique fields
            existing['fields'] = list({f['name']: f for f in existing['fields']}.values())
            existing.setdefault('type', 'struct')
    else:
        type = ""
        if t['kind'] == 'TypedefDecl':
            type = 'typedef'
        elif t['kind'] == 'EnumDecl':
            type = 'enum'
        elif t['kind'] == 'RecordDecl':
            type = 'struct'

        if type == "":
            print(f"Warning: Type {name} has unknown kind {t['kind']}, skipping.")
            continue

        generated_type = {
            'name': name,
            'type': type,
        }

        if type == 'struct':
            generated_type['fields'] = t.get('fields', [])
        elif type == 'enum':
            generated_type['enumerators'] = t.get('enumerators', [])

        unique_types[name] = generated_type

# --- Deduplicate functions by name ---
unique_functions = {}
for fn in all_functions:
    name = fn.get('name')
    if name and name not in unique_functions:
        unique_functions[name] = fn

# --- Deduplicate functions by name + arity (optional) ---
# Skipping deduplication here unless collisions are found

# --- Write output ---
with open(TYPES_OUT, 'w') as f:
    json.dump(sorted(unique_types.values(), key=lambda x: x['name']), f, indent=2)
print(f"✅ Wrote {len(unique_types)} unique types to {TYPES_OUT}")

with open(FUNCS_OUT, 'w') as f:
    json.dump(sorted(unique_functions.values(), key=lambda x: x['name']), f, indent=2)
print(f"✅ Wrote {len(all_functions)} functions to {FUNCS_OUT}")
