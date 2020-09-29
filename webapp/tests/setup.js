import '@babel/polyfill';
import Adapter from 'enzyme-adapter-react-16';
import {configure} from 'enzyme';

configure({adapter: new Adapter()});

global.ReactBootstrap = require('react-bootstrap');

global.PostUtils = {
    formatText: jest.fn().mockImplementation((t) => 'mockFormatText(' + t + ')'),
    messageHtmlToComponent: jest.fn().mockImplementation((t) => 'mockMessageHtmlToComponent(' + t + ')'),
};
