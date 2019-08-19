import React from 'react';
import {shallow} from 'enzyme';

import ActionButton from 'components/post_type/action_view/action_button/action_button';

describe('components/action_button/ActionButton', () => {
    const baseProps = {
        postId: 'post_id1',
        action: {
            id: 'action_id1',
            name: 'action_name',
        },
        hasVoted: false,
        actions: {
            voteAnswer: jest.fn(),
        },
    };
    test('should match snapshot', () => {
        const wrapper = shallow(<ActionButton {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot with hasVoted', () => {
        const newProps = {
            ...baseProps,
            hasVoted: true,
        };
        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
