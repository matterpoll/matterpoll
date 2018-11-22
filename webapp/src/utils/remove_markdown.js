import marked from 'marked';

export default class RemoveMarkdown extends marked.Renderer {
    code(text) {
        return text.replace(/\n/g, ' ');
    }

    blockquote(text) {
        return text.replace(/\n/g, ' ');
    }

    heading(text) {
        return text + ' ';
    }

    hr() {
        return '';
    }

    list(body) {
        return body;
    }

    listitem(text) {
        return text + ' ';
    }

    paragraph(text) {
        return text;
    }

    table() {
        return '';
    }

    tablerow() {
        return '';
    }

    tablecell() {
        return '';
    }

    strong(text) {
        return text;
    }

    em(text) {
        return text;
    }

    codespan(text) {
        return text.replace(/\n/g, ' ');
    }

    br() {
        return ' ';
    }

    del(text) {
        return text;
    }

    link(href, title, text) {
        return text;
    }

    image(href, title, text) {
        return text;
    }

    text(text) {
        return text.replace('\n', ' ');
    }
}
