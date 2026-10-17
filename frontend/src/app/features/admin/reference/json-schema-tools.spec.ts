import { CompletionContext } from '@codemirror/autocomplete';
import { json } from '@codemirror/lang-json';
import { ensureSyntaxTree } from '@codemirror/language';
import { EditorState } from '@codemirror/state';
import { JsonSchema, catalogProblems, compileSchema, nodeAt, pathAt, schemaCompletions, schemaProblems } from './json-schema-tools';

const SCHEMA: JsonSchema = {
  type: 'object',
  required: ['categories'],
  properties: {
    categories: {
      type: 'array',
      items: {
        type: 'object',
        required: ['key', 'icon', 'active'],
        properties: {
          key: { type: 'string', pattern: '^[a-z]+$', description: 'Stable key' },
          icon: { type: 'string', enum: ['hen', 'pig'], description: 'The drawing' },
          active: { type: 'boolean' }
        }
      }
    }
  }
};

const t = (key: string, params?: Record<string, string | number>) => `${key}${params ? ' ' + JSON.stringify(params) : ''}`;

function stateOf(text: string): EditorState {
  const state = EditorState.create({ doc: text, extensions: [json()] });
  ensureSyntaxTree(state, state.doc.length, 1e9);
  return state;
}

const DOC = `{
  "categories": [
    { "key": "poultry", "icon": "hen", "active": true },
    { "key": "swine", "icon": "dragon", "active": true }
  ]
}`;

describe('json-schema-tools', () => {
  it('finds the text of a path, a property name, or the nearest parent', () => {
    const state = stateOf(DOC);
    const slice = (n: { from: number; to: number } | null) => (n ? state.sliceDoc(n.from, n.to) : null);
    expect(slice(nodeAt(state, ['categories', 1, 'icon']))).toBe('"dragon"');
    expect(slice(nodeAt(state, ['categories', 1, 'icon'], true))).toBe('"icon"');
    expect(slice(nodeAt(state, ['categories', 1, 'missing']))!.startsWith('{ "key": "swine"')).toBeTrue();
  });

  it('knows the path at the cursor, and whether a name goes there', () => {
    const state = stateOf(DOC);
    const inIcon = pathAt(state, DOC.indexOf('dragon') + 2);
    expect(inIcon.path).toEqual(['categories', 1, 'icon']);
    expect(inIcon.inName).toBeFalse();
    const inName = pathAt(state, DOC.indexOf('"active": true }\n  ]') + 3);
    expect(inName.path).toEqual(['categories', 1]);
    expect(inName.inName).toBeTrue();
  });

  it('puts schema errors on the value at fault, in the reader’s words', () => {
    const state = stateOf(DOC);
    const errors = compileSchema(SCHEMA)(JSON.parse(DOC));
    const [d] = schemaProblems(state, errors, SCHEMA, t);
    expect(state.sliceDoc(d.from, d.to)).toBe('"dragon"');
    expect(d.message).toContain('adminReference.problem.enum');
    expect(d.message).toContain('hen, pig');
  });

  it('flags a key used twice and a published key that disappeared', () => {
    const doc = `{ "categories": [ { "key": "poultry", "subcategories": [ { "key": "poultry" } ] } ], "forumCategories": [] }`;
    const base = { categories: [{ key: 'poultry', subcategories: [] }], forumCategories: [{ key: 'health' }] };
    const problems = catalogProblems(stateOf(doc), JSON.parse(doc), base, t);
    expect(problems.filter((p) => p.message.includes('problem.duplicate')).length).toBe(2);
    expect(problems.some((p) => p.message.includes('problem.removed') && p.message.includes('health'))).toBeTrue();
  });

  it('completes the allowed values and the properties an object still lacks', () => {
    const complete = schemaCompletions(() => SCHEMA);
    const valueDoc = '{ "categories": [ { "key": "x", "icon": "po" } ] }';
    const value = complete(new CompletionContext(stateOf(valueDoc), valueDoc.indexOf('"po"') + 3, true));
    expect(value!.options.map((o) => o.label)).toEqual(['"hen"', '"pig"']);

    const nameDoc = '{ "categories": [ { "key": "x", "" } ] }';
    const names = complete(new CompletionContext(stateOf(nameDoc), nameDoc.indexOf('""') + 1, true));
    expect(names!.options.map((o) => o.label)).toEqual(['"icon"', '"active"']);
  });
});
