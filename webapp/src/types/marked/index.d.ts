/**
 * Minimal declarations for the Mattermost fork of marked pinned in package.json
 * (a fork of marked 0.3.x, which shipped no types of its own; the published
 * @types/marked describe a much later API).
 *
 * Only the renderer methods `@/utils/remove_markdown` overrides are declared, so that
 * those overrides are checked against the signatures marked actually calls them with.
 */
declare module 'marked' {
    type TableCellFlags = {
        header: boolean;
        align: 'center' | 'left' | 'right' | null;
    };

    class Renderer {
        code(code: string, lang?: string, escaped?: boolean): string;
        blockquote(quote: string): string;
        heading(text: string, level?: number, raw?: string): string;
        hr(): string;
        list(body: string, ordered?: boolean, start?: number): string;
        listitem(text: string): string;
        paragraph(text: string): string;
        table(header: string, body: string): string;
        tablerow(content: string): string;
        tablecell(content: string, flags?: TableCellFlags): string;
        strong(text: string): string;
        em(text: string): string;
        codespan(text: string): string;
        br(): string;
        del(text: string): string;
        link(href: string, title: string | null | undefined, text: string): string;
        image(href: string, title: string | null | undefined, text: string): string;
        text(text: string): string;
    }

    const marked: {
        Renderer: typeof Renderer;
    };

    export default marked;
}
