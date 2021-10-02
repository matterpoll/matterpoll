import 'mattermost-webapp/tests/setup';
import '@babel/polyfill';
import {jest} from '@jest/globals';

// @ts-ignore
global.ReactBootstrap = require('react-bootstrap');

// @ts-ignore
global.PostUtils = {
    formatText: jest.fn().mockImplementation((t) => 'mockFormatText(' + t + ')'),
    messageHtmlToComponent: jest.fn().mockImplementation((t) => 'mockMessageHtmlToComponent(' + t + ')'),
};
