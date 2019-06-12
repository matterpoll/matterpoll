import React from 'react';
import {shallow} from 'enzyme';

import ActionButton from 'components/post_type/action_view/action_button/action_button';

describe('components/action_button/ActionButton', () => {
    const baseProps = {
        currentUserId: 'user_id1',
        postId: 'post_id1',
        action: {
            id: 'action_id1',
            name: 'action_name',
        },
        voters: [
            'user_id1',
            'user_id2',
        ],
        actions: {doPostAction: jest.fn()},
    };
    test('should match snapshot', () => {
        const wrapper = shallow(<ActionButton {...baseProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('should match snapshot without voted', () => {
        const newProps = {
            ...baseProps,
            currentUserId: 'user_id3',
        };
        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});