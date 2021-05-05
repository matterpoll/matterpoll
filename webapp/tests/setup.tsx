// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import 'mattermost-webapp/tests/setup';
import '@babel/polyfill';

global.ReactBootstrap = require('react-bootstrap');
global.PostUtils = {
    formatText: jest.fn().mockImplementation((t) => 'mockFormatText(' + t + ')'),
    messageHtmlToComponent: jest.fn().mockImplementation((t) => 'mockMessageHtmlToComponent(' + t + ')'),
};
