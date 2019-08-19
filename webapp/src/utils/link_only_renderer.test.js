import LinkOnlyRenderer from './link_only_renderer';

describe('LinkOnlyRenderer', () => {
    it('success', () => {
        const href = 'http://example.com';
        const title = 'Title';
        const text = 'Sample Text';

        const expected = '<a class="theme markdown__link" href="http://example.com" target="_blank" title="Title">Sample Text</a>';

        expect(new LinkOnlyRenderer().link(href, title, text)).toEqual(expected);
    });
    it('success, with invalid url scheme', () => {
        const href = 'example.com';
        const title = 'Title';
        const text = 'Sample Text';

        const expected = '<a class="theme markdown__link" href="http://example.com" target="_blank" title="Title">Sample Text</a>';

        expect(new LinkOnlyRenderer().link(href, title, text)).toEqual(expected);
    });
    it('success, without title', () => {
        const href = 'example.com';
        const title = undefined; // eslint-disable-line no-undefined
        const text = 'Sample Text';

        const expected = '<a class="theme markdown__link" href="http://example.com" target="_blank">Sample Text</a>';

        expect(new LinkOnlyRenderer().link(href, title, text)).toEqual(expected);
    });
    it('success, with title having double quote', () => {
        const href = 'example.com';
        const title = 'Ti"tle';
        const text = 'Sample Text';

        const expected = '<a class="theme markdown__link" href="http://example.com" target="_blank" title="Ti"tle">Sample Text</a>';

        expect(new LinkOnlyRenderer().link(href, title, text)).toEqual(expected);
    });
});
