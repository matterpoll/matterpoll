import '@babel/polyfill';
import {jest} from '@jest/globals';

import Enzyme from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';

Enzyme.configure({adapter: new Adapter()});

// @ts-ignore
global.ReactBootstrap = require('react-bootstrap');

// @ts-ignore
global.PostUtils = {
    formatText: jest.fn().mockImplementation((t) => 'mockFormatText(' + t + ')'),
    messageHtmlToComponent: jest.fn().mockImplementation((t) => 'mockMessageHtmlToComponent(' + t + ')'),
};
