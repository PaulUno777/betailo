import { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete';
import { syntaxTree } from '@codemirror/language';
import { Diagnostic } from '@codemirror/lint';
import { EditorState } from '@codemirror/state';
import { SyntaxNode } from '@lezer/common';
import Ajv2020, { ErrorObject } from 'ajv/dist/2020';

/** A JSON Schema, as the API serves it (GET /admin/reference/schema). */
export interface JsonSchema {
  readonly type?: string;
  readonly description?: string;
  readonly enum?: readonly string[];
  readonly properties?: Readonly<Record<string, JsonSchema>>;
  readonly required?: readonly string[];
  readonly items?: JsonSchema;
  readonly pattern?: string;
  readonly minLength?: number;
  readonly maxLength?: number;
  readonly minItems?: number;
}

export type Path = readonly (string | number)[];

/** Translates a message key with params (LocaleService.t). */
export type Translate = (key: string, params?: Record<string, string | number>) => string;

// --- Where a path is in the text -------------------------------------

/** The syntax node of the value at path, or the deepest one found on
 * the way (a missing property points at its object). `name: true`
 * returns a property's name rather than its value. */
export function nodeAt(state: EditorState, path: Path, name = false): SyntaxNode | null {
  let node: SyntaxNode | null = syntaxTree(state).topNode.firstChild;
  let found = node;
  for (const [i, step] of path.entries()) {
    if (!node) break;
    if (node.name === 'Object' && typeof step === 'string') {
      const prop = node.getChildren('Property').find((p) => propertyName(state, p) === step);
      if (!prop) break;
      node = name && i === path.length - 1 ? prop.getChild('PropertyName') : prop.lastChild;
    } else if (node.name === 'Array' && typeof step === 'number') {
      node = arrayValues(node)[step] ?? null;
    } else {
      break;
    }
    if (node) found = node;
  }
  return found;
}

/** The path to the value at pos, and whether pos is where a property
 * name goes (inside a name, or in an object between properties); then
 * `object` is that object. */
export function pathAt(state: EditorState, pos: number): { path: (string | number)[]; inName: boolean; object: SyntaxNode | null } {
  const node = syntaxTree(state).resolveInner(pos, -1);
  if (node.name === 'PropertyName' && node.parent?.parent) {
    const object = node.parent.parent;
    return { path: pathOf(state, object), inName: true, object };
  }
  if (node.name === 'Object' || ((node.name === '{' || node.name === ',') && node.parent?.name === 'Object')) {
    const object = node.name === 'Object' ? node : node.parent!;
    return { path: pathOf(state, object), inName: true, object };
  }
  // Right after "name": with no value yet.
  const prop = node.name === 'Property' ? node : node.name === ':' ? node.parent : null;
  if (prop?.parent) {
    return { path: [...pathOf(state, prop.parent), propertyName(state, prop)], inName: false, object: null };
  }
  return { path: pathOf(state, node), inName: false, object: null };
}

/** The path from the document's root to node. */
export function pathOf(state: EditorState, node: SyntaxNode): (string | number)[] {
  const path: (string | number)[] = [];
  for (let c: SyntaxNode = node; c.parent; c = c.parent) {
    const parent = c.parent;
    if (parent.name === 'Property' && c.name !== 'PropertyName') {
      path.unshift(propertyName(state, parent));
    } else if (parent.name === 'Array') {
      const i = arrayValues(parent).findIndex((v) => v.from === c.from && v.to === c.to);
      if (i >= 0) path.unshift(i);
    }
  }
  return path;
}

function propertyName(state: EditorState, prop: SyntaxNode): string {
  const n = prop.getChild('PropertyName');
  if (!n) return '';
  try {
    return JSON.parse(state.sliceDoc(n.from, n.to));
  } catch {
    return state.sliceDoc(n.from + 1, n.to - 1);
  }
}

function arrayValues(array: SyntaxNode): SyntaxNode[] {
  const out: SyntaxNode[] = [];
  for (let c = array.firstChild; c; c = c.nextSibling) {
    if (!['[', ']', ',', '⚠'].includes(c.name)) out.push(c);
  }
  return out;
}

/** The part of the schema that describes path. */
export function schemaAt(schema: JsonSchema | null, path: Path): JsonSchema | null {
  let s: JsonSchema | null | undefined = schema;
  for (const step of path) {
    if (!s) return null;
    s = typeof step === 'number' ? s.items : s.properties?.[step];
  }
  return s ?? null;
}

// --- Checks ------------------------------------------------------------

/** Compiles the schema once; returns the checker. */
export function compileSchema(schema: JsonSchema): (value: unknown) => readonly ErrorObject[] {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  const validate = ajv.compile(schema as object);
  return (value) => (validate(value) ? [] : (validate.errors ?? []));
}

/** One problem in the text, with what the problems panel shows. */
export interface Problem extends Diagnostic {
  readonly line: number;
}

/** Schema errors as diagnostics in the reader's language. */
export function schemaProblems(state: EditorState, errors: readonly ErrorObject[], schema: JsonSchema, t: Translate): Diagnostic[] {
  const seen = new Set<string>();
  const out: Diagnostic[] = [];
  for (const e of errors) {
    const path = pointerPath(e.instancePath);
    const extra = e.params as Record<string, unknown>;
    const target = e.keyword === 'additionalProperties' ? nodeAt(state, [...path, String(extra['additionalProperty'])], true) : nodeAt(state, path);
    const from = target?.from ?? 0;
    const to = e.keyword === 'required' ? from + 1 : (target?.to ?? 0);
    const message = describe(e, schemaAt(schema, path), t);
    const id = `${from}:${message}`;
    if (seen.has(id)) continue;
    seen.add(id);
    out.push({ from, to: Math.max(to, from + 1), severity: 'error', message, source: 'schema' });
  }
  return out;
}

function describe(e: ErrorObject, at: JsonSchema | null, t: Translate): string {
  const p = e.params as Record<string, unknown>;
  switch (e.keyword) {
    case 'required':
      return t('adminReference.problem.required', { name: String(p['missingProperty']) });
    case 'additionalProperties':
      return t('adminReference.problem.unknownProperty', { name: String(p['additionalProperty']) });
    case 'enum':
      return t('adminReference.problem.enum', { values: (p['allowedValues'] as string[]).join(', ') });
    case 'type':
      return t('adminReference.problem.type', { type: t('adminReference.type.' + String(p['type'])) });
    case 'pattern':
      return t('adminReference.problem.pattern') + (at?.description ? ' ' + at.description : '');
    case 'minLength':
      return t('adminReference.problem.empty');
    case 'maxLength':
      return t('adminReference.problem.tooLong', { max: Number(p['limit']) });
    case 'minItems':
      return t('adminReference.problem.minItems', { min: Number(p['limit']) });
    default:
      return e.message ?? e.keyword;
  }
}

function pointerPath(pointer: string): (string | number)[] {
  return pointer
    .split('/')
    .slice(1)
    .map((s) => s.replace(/~1/g, '/').replace(/~0/g, '~'))
    .map((s) => (/^\d+$/.test(s) ? Number(s) : s));
}

/** The editable lists, loosely typed for the checks below. */
interface Lists {
  categories?: { key?: string; subcategories?: { key?: string }[] }[];
  countries?: { code?: string; regions?: { key?: string; cities?: { key?: string }[] }[] }[];
  [list: string]: unknown;
}

const TERM_LISTS = ['forumCategories', 'reviewTags', 'reportReasons', 'farmReviewChecks'] as const;

/** Every key and where it sits, the way the API counts uniqueness:
 * categories and kinds share one space, regions and towns are per
 * country, each vocabulary is its own. */
function keyEntries(l: Lists): { scope: string; key: string; path: (string | number)[] }[] {
  const out: { scope: string; key: string; path: (string | number)[] }[] = [];
  const put = (scope: string, key: unknown, path: (string | number)[]) => {
    if (typeof key === 'string') out.push({ scope, key, path });
  };
  (l.categories ?? []).forEach((c, i) => {
    put('species', c?.key, ['categories', i, 'key']);
    (c?.subcategories ?? []).forEach((s, j) => put('species', s?.key, ['categories', i, 'subcategories', j, 'key']));
  });
  (l.countries ?? []).forEach((k, i) => {
    put('country', k?.code, ['countries', i, 'code']);
    (k?.regions ?? []).forEach((r, j) => {
      put(`region:${k?.code}`, r?.key, ['countries', i, 'regions', j, 'key']);
      (r?.cities ?? []).forEach((c, m) => put(`city:${k?.code}`, c?.key, ['countries', i, 'regions', j, 'cities', m, 'key']));
    });
  });
  for (const list of TERM_LISTS) {
    const terms = (l[list] as { key?: string }[] | undefined) ?? [];
    terms.forEach((term, i) => put(list, term?.key, [list, i, 'key']));
  }
  // One vocabulary edited on its own: {"entries": [...]}.
  ((l['entries'] as { key?: string }[] | undefined) ?? []).forEach((term, i) => put('entries', term?.key, ['entries', i, 'key']));
  return out;
}

/** The rules the schema can't say, checked before the API does: a key
 * used twice, and a published key that disappeared (an entry in use is
 * deactivated, never removed). */
export function catalogProblems(state: EditorState, value: unknown, base: unknown, t: Translate): Diagnostic[] {
  const out: Diagnostic[] = [];
  if (!value || typeof value !== 'object') return out;
  const mine = keyEntries(value as Lists);
  const seen = new Map<string, number>();
  for (const k of mine) {
    const id = `${k.scope}|${k.key}`;
    seen.set(id, (seen.get(id) ?? 0) + 1);
  }
  for (const k of mine) {
    if ((seen.get(`${k.scope}|${k.key}`) ?? 0) > 1) {
      const n = nodeAt(state, k.path);
      if (n) out.push({ from: n.from, to: n.to, severity: 'error', message: t('adminReference.problem.duplicate', { key: k.key }), source: 'keys' });
    }
  }
  if (!base || typeof base !== 'object') return out;
  const kept = new Set(mine.map((k) => `${k.scope}|${k.key}`));
  const removed = keyEntries(base as Lists).filter((k) => !kept.has(`${k.scope}|${k.key}`));
  for (const k of removed) {
    // Point at the list the entry was in.
    const n = nodeAt(state, k.path.slice(0, -2));
    out.push({
      from: n?.from ?? 0,
      to: (n?.from ?? 0) + 1,
      severity: 'error',
      message: t('adminReference.problem.removed', { key: k.key }),
      source: 'keys'
    });
  }
  return out;
}

// --- Help while typing --------------------------------------------------

/** Completes property names (the ones the object may still take) and
 * values the schema lists (icons, colours, cycle types, true/false). */
export function schemaCompletions(schema: () => JsonSchema | null) {
  return (context: CompletionContext): CompletionResult | null => {
    const root = schema();
    if (!root) return null;
    const { path, inName, object } = pathAt(context.state, context.pos);
    const word = context.matchBefore(/"?[\w-]*/);
    if (!word || (word.from === word.to && !context.explicit)) return null;
    const from = word.from;

    if (inName) {
      const objectSchema = schemaAt(root, path);
      const props = objectSchema?.properties;
      if (!props) return null;
      const present = new Set(object ? object.getChildren('Property').map((p) => propertyName(context.state, p)) : []);
      const options: Completion[] = Object.entries(props)
        .filter(([name]) => !present.has(name))
        .map(([name, s]) => ({
          label: `"${name}"`,
          type: 'property',
          detail: objectSchema?.required?.includes(name) ? '*' : undefined,
          info: s.description,
          apply: `"${name}": ${emptyValue(s)}`
        }));
      return { from, options, validFor: /^"?[\w-]*$/ };
    }

    const valueSchema = schemaAt(root, path);
    if (valueSchema?.enum) {
      return {
        from,
        options: valueSchema.enum.map((v) => ({ label: `"${v}"`, type: 'enum', info: valueSchema.description })),
        validFor: /^"?[\w-]*$/
      };
    }
    if (valueSchema?.type === 'boolean') {
      return { from, options: [{ label: 'true', type: 'keyword' }, { label: 'false', type: 'keyword' }] };
    }
    return null;
  };
}

function emptyValue(s: JsonSchema): string {
  switch (s.type) {
    case 'object':
      return '{}';
    case 'array':
      return '[]';
    case 'boolean':
      return 'true';
    default:
      return '""';
  }
}
