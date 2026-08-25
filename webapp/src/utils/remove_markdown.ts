import marked from 'marked';

export default class RemoveMarkdown extends marked.Renderer {
    code(text: string) {
        return text.replace(/\n/g, ' ');
    }

    blockquote(text: string) {
        return text.replace(/\n/g, ' ');
    }

    heading(text: string) {
        return text + ' ';
    }

    hr() {
        return '';
    }

    list(body: string) {
        return body;
    }

    listitem(text: string) {
        return text + ' ';
    }

    paragraph(text: string) {
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

    strong(text: string) {
        return text;
    }

    em(text: string) {
        return text;
    }

    codespan(text: string) {
        return text.replace(/\n/g, ' ');
    }

    br() {
        return ' ';
    }

    del(text: string) {
        return text;
    }

    link(href: string, title: string | null | undefined, text: string) {
        return text;
    }

    image(href: string, title: string | null | undefined, text: string) {
        return text;
    }

    text(text: string) {
        return text.replace('\n', ' ');
    }
}
