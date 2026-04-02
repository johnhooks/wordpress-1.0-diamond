import { Schema, Node } from "prosemirror-model"
import { schema as basicSchema } from "prosemirror-schema-basic"
import { addListNodes } from "prosemirror-schema-list"
import { EditorState } from "prosemirror-state"
import { EditorView } from "prosemirror-view"
import { history, undo, redo } from "prosemirror-history"
import { keymap } from "prosemirror-keymap"
import { baseKeymap, toggleMark } from "prosemirror-commands"

const schema = new Schema({
  nodes: addListNodes(basicSchema.spec.nodes, "paragraph block*", "block"),
  marks: basicSchema.spec.marks,
})

const plugins = [
  history(),
  keymap({
    "Mod-z": undo,
    "Mod-y": redo,
    "Mod-Shift-z": redo,
    "Mod-b": toggleMark(schema.marks.strong),
    "Mod-i": toggleMark(schema.marks.em),
    "Mod-`": toggleMark(schema.marks.code),
  }),
  keymap(baseKeymap),
]

class TheEditor extends HTMLElement {
  connectedCallback() {
    const id = this.getAttribute("for")
    const textarea = document.getElementById(id)
    if (!textarea) {
      this.textContent = "Editor error: no textarea found."
      return
    }

    let doc
    try {
      const json = JSON.parse(textarea.value)
      doc = Node.fromJSON(schema, json)
    } catch {
      doc = schema.node("doc", null, [schema.node("paragraph")])
    }

    const style = document.createElement("style")
    style.textContent = `
      .ProseMirror { outline: none; min-height: 20em; }
      .ProseMirror p:first-child { margin-top: 0; }
      .ProseMirror p:last-child { margin-bottom: 0; }
    `
    this.appendChild(style)

    const editorDiv = document.createElement("div")
    editorDiv.style.cssText = "border: 1px solid #ccc; padding: 0.5em;"
    this.appendChild(editorDiv)

    const state = EditorState.create({ doc, plugins })
    this.view = new EditorView(editorDiv, {
      state,
      dispatchTransaction: (tr) => {
        const newState = this.view.state.apply(tr)
        this.view.updateState(newState)
        textarea.value = JSON.stringify(newState.doc.toJSON())
      },
    })
  }
}

customElements.define("the-editor", TheEditor)
