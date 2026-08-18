import assert from 'node:assert/strict';
import { CodeHighlightNode, CodeNode } from '@lexical/code';
import { AutoLinkNode, LinkNode } from '@lexical/link';
import {
  $convertFromMarkdownString,
  $convertToMarkdownString,
  ELEMENT_TRANSFORMERS,
  TEXT_FORMAT_TRANSFORMERS,
  TEXT_MATCH_TRANSFORMERS,
} from '@lexical/markdown';
import {
  $insertList,
  $isListNode,
  $removeList,
  ListItemNode,
  ListNode,
} from '@lexical/list';
import { HeadingNode, QuoteNode } from '@lexical/rich-text';
import { TableCellNode, TableNode, TableRowNode } from '@lexical/table';
import {
  $createParagraphNode,
  $createTextNode,
  $getRoot,
  createEditor,
} from 'lexical';

const nodes = [
  HeadingNode,
  ListNode,
  ListItemNode,
  QuoteNode,
  CodeNode,
  CodeHighlightNode,
  LinkNode,
  AutoLinkNode,
  TableNode,
  TableRowNode,
  TableCellNode,
];
const transformers = [
  ...ELEMENT_TRANSFORMERS,
  ...TEXT_FORMAT_TRANSFORMERS,
  ...TEXT_MATCH_TRANSFORMERS,
];
const source = [
  '## Editor regression',
  '',
  '- first item',
  '- second item',
  '',
  '> quoted text',
  '',
  '[Lexical](https://lexical.dev)',
  '',
  '```ts',
  'const version = 49;',
  '```',
].join('\n');

const createTestEditor = () =>
  createEditor({
    namespace: 'lexical-regression',
    nodes,
    onError: error => {
      throw error;
    },
  });

const editor = createTestEditor();
editor.update(
  () => {
    $convertFromMarkdownString(source, transformers);
  },
  { discrete: true }
);

const markdown = editor
  .getEditorState()
  .read(() => $convertToMarkdownString(transformers));
const text = editor.getEditorState().read(() => $getRoot().getTextContent());
assert.match(markdown, /## Editor regression/);
assert.match(markdown, /- first item/);
assert.match(markdown, /https:\/\/lexical\.dev/);
assert.match(markdown, /const version = 49;/);
assert.match(text, /quoted text/);

const serialized = JSON.stringify(editor.toJSON().editorState);
const restoredEditor = createTestEditor();
restoredEditor.setEditorState(restoredEditor.parseEditorState(serialized));
assert.equal(
  restoredEditor.getEditorState().read(() => $getRoot().getTextContent()),
  text
);

const listEditor = createTestEditor();
listEditor.update(
  () => {
    const paragraph = $createParagraphNode();
    paragraph.append($createTextNode('List API compatibility'));
    $getRoot().append(paragraph);
    paragraph.selectEnd();
    $insertList('bullet');
    assert.equal($isListNode($getRoot().getFirstChild()), true);
    $removeList();
  },
  { discrete: true }
);
assert.equal(
  listEditor.getEditorState().read(() => $getRoot().getTextContent()),
  'List API compatibility'
);

console.log('Lexical 0.49 editor regression passed');
