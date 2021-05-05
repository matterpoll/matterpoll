import React from 'react';
import {shallow} from 'enzyme';

import {Constants} from 'mattermost-webapp/utils/constants';

import ActionButton from 'components/post_type/action_view/action_button/action_button';

describe('components/action_button/ActionButton', () => {
    const baseProps = {
        postId: 'post_id1',
        action: {
            id: 'action_id1',
            name: 'action_name',
        },
        theme: Constants.THEMES.default,
        hasVoted: false,
        actions: {
            voteAnswer: jest.fn(),
        },
    };
    test('without style should match snapshot', () => {
        const wrapper = shallow(<ActionButton {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('without style with hasVoted should match snapshot', () => {
        const newProps = {
            ...baseProps,
            hasVoted: true,
        };
        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
