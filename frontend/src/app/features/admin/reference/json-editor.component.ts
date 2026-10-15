import { ChangeDetectionStrategy, Component, ElementRef, OnDestroy, afterNextRender, effect, inject, input, output, untracked } from '@angular/core';
import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { json, jsonParseLinter } from '@codemirror/lang-json';
import { HighlightStyle, bracketMatching, foldGutter, foldKeymap, indentOnInput, syntaxHighlighting, syntaxTree } from '@codemirror/language';
import { Diagnostic, forceLinting, lintGutter, linter } from '@codemirror/lint';
import { highlightSelectionMatches, searchKeymap } from '@codemirror/search';
import { EditorState } from '@codemirror/state';
import { EditorView, drawSelection, highlightActiveLine, highlightActiveLineGutter, hoverTooltip, keymap, lineNumbers } from '@codemirror/view';
import { tags } from '@lezer/highlight';
import { LocaleService } from '../../../core/i18n/locale.service';
import { ANIMAL_PATHS, isAnimalName } from '../../../shared/ui/animal-icon/animal-paths';
import { JsonSchema, Problem, catalogProblems, compileSchema, pathOf, schemaAt, schemaCompletions, schemaProblems } from './json-schema-tools';

/** Colours from the design tokens, so the editor follows the theme. */
const highlight = HighlightStyle.define([
  { tag: tags.propertyName, color: 'var(--color-forest)', fontWeight: '600' },
  { tag: tags.string, color: 'var(--color-clay-ink)' },
  { tag: tags.number, color: 'var(--color-sky)' },
  { tag: [tags.bool, tags.null], color: 'var(--color-plum)', fontWeight: '600' },
  { tag: [tags.brace, tags.squareBracket, tags.separator, tags.punctuation], color: 'var(--color-ink-muted)' }
]);

const theme = EditorView.theme({
  '&': { height: '100%', fontSize: '13px', backgroundColor: 'var(--color-surface-card)', color: 'var(--color-ink)' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
  '.cm-content': { caretColor: 'var(--color-ink)' },
  '.cm-cursor': { borderLeftColor: 'var(--color-ink)' },
  '.cm-gutters': { backgroundColor: 'var(--color-surface-sunken)', color: 'var(--color-ink-muted)', border: 'none' },
  '.cm-activeLine': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 7%, transparent)' },
  '.cm-activeLineGutter': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 14%, transparent)', color: 'var(--color-ink)' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 22%, transparent) !important' },
  '.cm-matchingBracket': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 20%, transparent)', outline: 'none' },
  '.cm-tooltip': { backgroundColor: 'var(--color-surface-card)', color: 'var(--color-ink)', border: '1px solid var(--color-line)', borderRadius: '10px', boxShadow: 'var(--shadow-float)', overflow: 'hidden' },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': { backgroundColor: 'var(--color-forest)', color: 'var(--color-on-forest)' },
  '.cm-completionInfo': { maxWidth: '320px', padding: '8px 10px', fontFamily: 'var(--font-sans, inherit)', fontSize: '12.5px' },
  '.cm-tooltip-lint': { padding: '0' },
  '.cm-diagnostic': { fontFamily: 'var(--font-sans, inherit)', fontSize: '13px', padding: '6px 10px' },
  '.cm-panels': { backgroundColor: 'var(--color-surface-sunken)', color: 'var(--color-ink)', borderColor: 'var(--color-line)' },
  '.cm-panel input, .cm-panel button': { fontSize: '13px' },
  '.bt-hover': { maxWidth: '320px', padding: '8px 10px', fontSize: '12.5px', lineHeight: '1.45', display: 'flex', gap: '10px', alignItems: 'center' },
  '.bt-hover svg': { flexShrink: '0', color: 'var(--color-forest)' }
});

/** The reference data as JSON, edited like a cloud policy: highlighted,
 * folded, searched (Ctrl/Cmd-F), checked against the API's schema as
 * you type (underlined at the place, in the gutter and in `problems`),
 * with completion of property names and allowed values (Ctrl-Space)
 * and each property's meaning on hover. */
@Component({
  selector: 'bt-json-editor',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: ''
})
export class JsonEditorComponent implements OnDestroy {
  private readonly host = inject(ElementRef<HTMLElement>);
  private readonly locale = inject(LocaleService);

  /** The text to start from; setting it again replaces the document. */
  readonly doc = input.required<string>();
  readonly schema = input<JsonSchema | null>(null);
  /** The version in force, to warn when a published key disappears. */
  readonly base = input<unknown>(null);
  readonly label = input('');

  readonly textChange = output<string>();
  readonly problems = output<Problem[]>();

  private view?: EditorView;
  private check: ReturnType<typeof compileSchema> | null = null;

  constructor() {
    afterNextRender(() => this.mount());
    effect(() => {
      const schema = this.schema();
      this.check = schema ? compileSchema(schema) : null;
      if (this.view) forceLinting(this.view);
    });
    effect(() => {
      const doc = this.doc();
      untracked(() => {
        if (this.view && doc !== this.view.state.doc.toString()) this.replace(doc);
      });
    });
    // Messages follow the language.
    effect(() => {
      this.locale.locale();
      if (this.view) forceLinting(this.view);
    });
  }

  ngOnDestroy(): void {
    this.view?.destroy();
  }

  /** Starts over from text (Revert, a restored draft, a new version). */
  reset(text: string): void {
    if (this.view && text !== this.view.state.doc.toString()) this.replace(text);
  }

  /** Re-indents the document (2 spaces), when it parses. */
  format(): void {
    if (!this.view) return;
    try {
      const pretty = JSON.stringify(JSON.parse(this.view.state.doc.toString()), null, 2);
      this.replace(pretty, true);
    } catch {
      /* not JSON yet: the problems say where */
    }
  }

  /** Puts text at the cursor (or over the selection). */
  insert(text: string): void {
    if (!this.view) return;
    this.view.dispatch(this.view.state.replaceSelection(text));
    this.view.focus();
  }

  /** Selects a problem's place and brings it into view. */
  goTo(from: number, to = from): void {
    if (!this.view) return;
    this.view.dispatch({ selection: { anchor: from, head: to }, scrollIntoView: true });
    this.view.focus();
  }

  private replace(text: string, keepHistory = false): void {
    const view = this.view!;
    if (keepHistory) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
    } else {
      view.setState(this.state(text));
    }
  }

  private mount(): void {
    this.view = new EditorView({ state: this.state(this.doc()), parent: this.host.nativeElement });
  }

  private state(doc: string): EditorState {
    const t = (key: string, params?: Record<string, string | number>) => this.locale.t(key, params);
    const parseLint = jsonParseLinter();
    return EditorState.create({
      doc,
      extensions: [
        lineNumbers(),
        foldGutter(),
        lintGutter(),
        highlightActiveLineGutter(),
        highlightActiveLine(),
        drawSelection(),
        history(),
        indentOnInput(),
        bracketMatching(),
        closeBrackets(),
        highlightSelectionMatches(),
        json(),
        syntaxHighlighting(highlight),
        theme,
        EditorView.contentAttributes.of({ 'aria-label': this.label() }),
        EditorState.tabSize.of(2),
        keymap.of([...closeBracketsKeymap, ...defaultKeymap, ...searchKeymap, ...historyKeymap, ...foldKeymap, ...completionKeymap, indentWithTab]),
        autocompletion({ override: [schemaCompletions(() => this.schema())], icons: false }),
        linter(
          (view) => {
            const syntax = parseLint(view);
            let diagnostics: Diagnostic[] = syntax.map((d) => ({ ...d, message: t('adminReference.problem.syntax', { detail: d.message }) }));
            if (!syntax.length) {
              const value = JSON.parse(view.state.doc.toString());
              const schema = this.schema();
              if (schema && this.check) diagnostics = schemaProblems(view.state, this.check(value), schema, t);
              diagnostics = [...diagnostics, ...catalogProblems(view.state, value, this.base(), t)];
            }
            this.problems.emit(
              diagnostics
                .map((d) => ({ ...d, line: view.state.doc.lineAt(Math.min(d.from, view.state.doc.length)).number }))
                .sort((a, b) => a.from - b.from)
            );
            return diagnostics;
          },
          { delay: 250 }
        ),
        hoverTooltip((view, pos, side) => this.hover(view, pos, side)),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) this.textChange.emit(u.state.doc.toString());
        })
      ]
    });
  }

  /** On a property name: what it means. On an icon value: the drawing. */
  private hover(view: EditorView, pos: number, side: -1 | 1) {
    const node = syntaxTree(view.state).resolveInner(pos, side);
    const prop = node.name === 'PropertyName' ? node.parent : node.name === 'String' && node.parent?.name === 'Property' ? node.parent : null;
    if (!prop) return null;
    const nameNode = prop.getChild('PropertyName');
    if (!nameNode) return null;
    const name = view.state.sliceDoc(nameNode.from + 1, nameNode.to - 1);
    const value = node.name === 'String' ? view.state.sliceDoc(node.from + 1, node.to - 1) : null;
    const described = schemaAt(this.schema(), [...pathOf(view.state, prop.parent!), name]);
    const icon = name === 'icon' && value && isAnimalName(value) ? ANIMAL_PATHS[value] : null;
    if (!described?.description && !icon) return null;
    return {
      pos: node.from,
      end: node.to,
      above: true,
      create: () => {
        const dom = document.createElement('div');
        dom.className = 'bt-hover';
        if (icon) {
          const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
          for (const [k, v] of Object.entries({ width: '40', height: '40', viewBox: '0 0 64 64', fill: 'none', stroke: 'currentColor', 'stroke-width': '2.5', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })) {
            svg.setAttribute(k, v);
          }
          // The drawing comes from this build's own constants.
          svg.innerHTML = icon;
          dom.appendChild(svg);
        }
        const text = document.createElement('span');
        text.textContent = value && icon ? value : (described?.description ?? '');
        dom.appendChild(text);
        return { dom };
      }
    };
  }
}
