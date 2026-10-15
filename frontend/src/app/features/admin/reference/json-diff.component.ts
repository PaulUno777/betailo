import { ChangeDetectionStrategy, Component, ElementRef, OnDestroy, afterNextRender, effect, inject, input } from '@angular/core';
import { json } from '@codemirror/lang-json';
import { unifiedMergeView } from '@codemirror/merge';
import { EditorState } from '@codemirror/state';
import { EditorView, lineNumbers } from '@codemirror/view';

const theme = EditorView.theme({
  '&': { fontSize: '13px', backgroundColor: 'var(--color-surface-card)', color: 'var(--color-ink)' },
  '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.6' },
  '.cm-gutters': { backgroundColor: 'var(--color-surface-sunken)', color: 'var(--color-ink-muted)', border: 'none' },
  '.cm-changedLine': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 12%, transparent) !important' },
  '.cm-insertedLine, .cm-changedText': { backgroundColor: 'color-mix(in srgb, var(--color-forest) 22%, transparent) !important' },
  '.cm-deletedChunk': { backgroundColor: 'color-mix(in srgb, var(--color-alert) 12%, transparent) !important' },
  '.cm-deletedChunk del, .cm-deletedText': { backgroundColor: 'color-mix(in srgb, var(--color-alert) 25%, transparent) !important', textDecoration: 'none' },
  '.cm-collapsedLines': { backgroundColor: 'var(--color-surface-sunken)', color: 'var(--color-ink-muted)', fontFamily: 'var(--font-sans, inherit)' }
});

/** The edits against the version in force, line by line: removed lines
 * in red, added and changed ones in green, the unchanged stretches
 * folded. Read-only: the review step before publishing. */
@Component({
  selector: 'bt-json-diff',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: ''
})
export class JsonDiffComponent implements OnDestroy {
  private readonly host = inject(ElementRef<HTMLElement>);

  readonly original = input.required<string>();
  readonly changed = input.required<string>();

  private view?: EditorView;

  constructor() {
    afterNextRender(() => this.render());
    effect(() => {
      this.original();
      this.changed();
      if (this.view) this.render();
    });
  }

  ngOnDestroy(): void {
    this.view?.destroy();
  }

  private render(): void {
    this.view?.destroy();
    this.view = new EditorView({
      parent: this.host.nativeElement,
      state: EditorState.create({
        doc: this.changed(),
        extensions: [
          lineNumbers(),
          json(),
          theme,
          EditorState.readOnly.of(true),
          EditorView.editable.of(false),
          unifiedMergeView({ original: this.original(), mergeControls: false, gutter: true, collapseUnchanged: { margin: 3, minSize: 6 } })
        ]
      })
    });
  }
}
