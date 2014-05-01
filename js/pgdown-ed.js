var editor = require("pagedown-editor");

function getPagedownEditor() {
    return editor.getPagedownEditor();
}

global.window.getPagedownEditor = getPagedownEditor;
