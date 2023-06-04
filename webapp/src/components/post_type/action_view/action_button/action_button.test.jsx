import React from 'react';
import {shallow} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';

import ActionButton from 'components/post_type/action_view/action_button/action_button';

describe('components/action_button/ActionButton', () => {
    const baseProps = {
        postId: 'post_id1',
        action: {
            id: 'action_id1',
            name: 'action_name',
        },
        theme: Preferences.THEMES.denim,
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
    test('without style with invalid theme should match snapshot', () => {
        const newProps = baseProps;
        newProps.theme = {};

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('with default style should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'default';

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('with default style with hasVoted should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'default';
        newProps.hasVoted = true;

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('with primary style should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'primary';

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
    test('with danger style should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'danger';

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('with invalid style should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'invalid_style_value';

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });

    test('with invalid theme should match snapshot', () => {
        const newProps = baseProps;
        newProps.action.style = 'default';
        newProps.theme = {};

        const wrapper = shallow(<ActionButton {...newProps}/>);
        expect(wrapper).toMatchSnapshot();
    });
});
