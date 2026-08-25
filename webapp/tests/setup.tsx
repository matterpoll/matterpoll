import '@testing-library/jest-dom';
import {jest} from '@jest/globals';

// The webapp exposes these helpers on `window.PostUtils` for plugins to reuse.
// Stub them so component tests assert on the plugin's own markup rather than on
// the webapp's markdown rendering.
window.PostUtils = {
    formatText: jest.fn((t: string) => 'mockFormatText(' + t + ')'),
    messageHtmlToComponent: jest.fn((t: string) => 'mockMessageHtmlToComponent(' + t + ')'),
};
